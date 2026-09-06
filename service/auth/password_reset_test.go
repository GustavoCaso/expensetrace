package auth

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/GustavoCaso/expensetrace/testutil"
)

func TestRequestPasswordReset_KnownUser(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	token, expiresAt, err := svc.RequestPasswordReset(context.Background(), user.Username())
	if err != nil {
		t.Fatalf("RequestPasswordReset returned unexpected error: %v", err)
	}

	if token == "" {
		t.Fatal("Expected non-empty token")
	}

	if !expiresAt.After(time.Now()) {
		t.Fatal("Expected expiresAt to be in the future")
	}

	stored, getErr := s.GetPasswordResetToken(context.Background(), token)
	if getErr != nil {
		t.Fatalf("Failed to get stored token: %v", getErr)
	}

	if stored.UserID() != user.ID() {
		t.Errorf("Expected token user id %d, got %d", user.ID(), stored.UserID())
	}
}

func TestRequestPasswordReset_UnknownUser(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	token, _, err := svc.RequestPasswordReset(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("RequestPasswordReset returned unexpected error: %v", err)
	}

	if token != "" {
		t.Fatal("Expected empty token for unknown user")
	}
}

func TestResetPassword_UpdatesPasswordAndInvalidatesSessions(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	sessionID, _, signinValidationErr, signinErr := svc.Signin(context.Background(), user.Username(), "test")
	if signinErr != nil || signinValidationErr != nil {
		t.Fatalf("Signin failed: %v %v", signinErr, signinValidationErr)
	}

	token, _, reqErr := svc.RequestPasswordReset(context.Background(), user.Username())
	if reqErr != nil {
		t.Fatalf("RequestPasswordReset returned unexpected error: %v", reqErr)
	}

	validationErr, err := svc.ResetPassword(context.Background(), token, "newpassword123", "newpassword123")
	if err != nil {
		t.Fatalf("ResetPassword returned unexpected error: %v", err)
	}
	if validationErr != nil {
		t.Fatalf("ResetPassword returned unexpected validation error: %v", validationErr)
	}

	updatedUser, getErr := s.GetUserByUsername(context.Background(), user.Username())
	if getErr != nil {
		t.Fatalf("Failed to get user: %v", getErr)
	}

	if compareErr := bcrypt.CompareHashAndPassword(
		[]byte(updatedUser.PasswordHash()), []byte("newpassword123"),
	); compareErr != nil {
		t.Error("Password should have been updated")
	}

	if _, sessionErr := s.GetSession(context.Background(), sessionID); sessionErr == nil {
		t.Error("Expected prior session to be invalidated after password reset")
	}
}

func TestResetPassword_RejectsMismatchedPasswords(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	token, _, reqErr := svc.RequestPasswordReset(context.Background(), user.Username())
	if reqErr != nil {
		t.Fatalf("RequestPasswordReset returned unexpected error: %v", reqErr)
	}

	validationErr, err := svc.ResetPassword(context.Background(), token, "newpassword123", "different123")
	if err != nil {
		t.Fatalf("ResetPassword returned unexpected internal error: %v", err)
	}
	if validationErr == nil {
		t.Fatal("Expected validationErr for mismatched passwords")
	}
}

func TestResetPassword_RejectsUnknownToken(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	validationErr, err := svc.ResetPassword(context.Background(), "unknown-token", "newpassword123", "newpassword123")
	if err != nil {
		t.Fatalf("ResetPassword returned unexpected internal error: %v", err)
	}
	if validationErr == nil {
		t.Fatal("Expected validationErr for unknown token")
	}
}

func TestResetPassword_InvalidatesOtherOutstandingTokens(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	oldToken, _, reqErr1 := svc.RequestPasswordReset(context.Background(), user.Username())
	if reqErr1 != nil {
		t.Fatalf("first RequestPasswordReset returned unexpected error: %v", reqErr1)
	}

	newToken, _, reqErr2 := svc.RequestPasswordReset(context.Background(), user.Username())
	if reqErr2 != nil {
		t.Fatalf("second RequestPasswordReset returned unexpected error: %v", reqErr2)
	}

	validationErr, err := svc.ResetPassword(context.Background(), newToken, "newpassword123", "newpassword123")
	if err != nil {
		t.Fatalf("ResetPassword returned unexpected error: %v", err)
	}
	if validationErr != nil {
		t.Fatalf("ResetPassword returned unexpected validation error: %v", validationErr)
	}

	// The older, still-unused token must now be invalidated too.
	oldValidationErr, oldErr := svc.ResetPassword(
		context.Background(),
		oldToken,
		"anotherpassword123",
		"anotherpassword123",
	)
	if oldErr != nil {
		t.Fatalf("ResetPassword with old token returned unexpected internal error: %v", oldErr)
	}
	if oldValidationErr == nil {
		t.Fatal("Expected validationErr for previously-outstanding token after another token was used")
	}

	storedOldToken, getErr := s.GetPasswordResetToken(context.Background(), oldToken)
	if getErr != nil {
		t.Fatalf("Failed to get old token: %v", getErr)
	}
	if storedOldToken.UsedAt() == nil {
		t.Error("Expected old outstanding token to be marked used once another token was consumed")
	}
}

func TestResetPassword_RejectsAlreadyUsedToken(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	token, _, reqErr := svc.RequestPasswordReset(context.Background(), user.Username())
	if reqErr != nil {
		t.Fatalf("RequestPasswordReset returned unexpected error: %v", reqErr)
	}

	if _, err := svc.ResetPassword(context.Background(), token, "newpassword123", "newpassword123"); err != nil {
		t.Fatalf("first ResetPassword returned unexpected error: %v", err)
	}

	validationErr, err := svc.ResetPassword(context.Background(), token, "anotherpassword123", "anotherpassword123")
	if err != nil {
		t.Fatalf("ResetPassword returned unexpected internal error: %v", err)
	}
	if validationErr == nil {
		t.Fatal("Expected validationErr for already-used token")
	}
}
