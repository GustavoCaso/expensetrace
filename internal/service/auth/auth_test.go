package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/domain"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestSignup_CreatesUserSessionAndExcludeCategory(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	sessionID, expiresAt, validationErr, err := svc.Signup(
		context.Background(),
		"newuser",
		"password123",
		"password123",
	)
	if err != nil {
		t.Fatalf("Signup returned unexpected error: %v", err)
	}
	if validationErr != nil {
		t.Fatalf("Signup returned unexpected validation error: %v", validationErr)
	}

	if sessionID == "" {
		t.Fatal("Expected non-empty sessionID")
	}

	if expiresAt.IsZero() {
		t.Fatal("Expected non-zero expiresAt")
	}

	user, getErr := s.GetUserByUsername(context.Background(), "newuser")
	if getErr != nil {
		t.Fatalf("Failed to get created user: %v", getErr)
	}

	categories, catErr := s.GetCategories(context.Background(), user.ID())
	if catErr != nil {
		t.Fatalf("Failed to get categories: %v", catErr)
	}

	found := false
	for _, c := range categories {
		if c.Name() == domain.ExcludeCategory {
			found = true
		}
	}
	if !found {
		t.Fatal("Expected Exclude category to be created for new user")
	}
}

func TestSignup_RejectsMismatchedPasswords(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	sessionID, _, validationErr, err := svc.Signup(context.Background(), "newuser", "password123", "different123")
	if err != nil {
		t.Fatalf("Signup returned unexpected internal error: %v", err)
	}

	if validationErr == nil {
		t.Fatal("Expected validationErr for mismatched passwords")
	}

	if sessionID != "" {
		t.Fatal("Expected empty sessionID on validation failure")
	}

	_, getErr := s.GetUserByUsername(context.Background(), "newuser")
	if getErr == nil {
		t.Fatal("Expected user to not be created on validation failure")
	}
}

func TestSignin_RejectsWrongPassword(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	sessionID, _, validationErr, err := svc.Signin(context.Background(), user.Username(), "wrongpassword")
	if err != nil {
		t.Fatalf("Signin returned unexpected internal error: %v", err)
	}

	if validationErr == nil {
		t.Fatal("Expected validationErr for wrong password")
	}

	const expectedMsg = "Invalid username or password"
	if validationErr.Error() != expectedMsg {
		t.Fatalf("Expected validationErr message %q, got %q", expectedMsg, validationErr.Error())
	}

	if sessionID != "" {
		t.Fatal("Expected empty sessionID on validation failure")
	}
}

func TestAuthenticatedUser_ReturnsSessionOwner(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	sessionID, _, validationErr, err := svc.Signin(context.Background(), user.Username(), "test")
	if err != nil {
		t.Fatalf("Signin returned unexpected error: %v", err)
	}
	if validationErr != nil {
		t.Fatalf("Signin returned unexpected validation error: %v", validationErr)
	}

	authUser, authErr := svc.AuthenticatedUser(context.Background(), sessionID)
	if authErr != nil {
		t.Fatalf("AuthenticatedUser returned unexpected error: %v", authErr)
	}

	if authUser.ID() != user.ID() {
		t.Errorf("Expected user ID %d, got %d", user.ID(), authUser.ID())
	}
	if authUser.Username() != user.Username() {
		t.Errorf("Expected username %q, got %q", user.Username(), authUser.Username())
	}
}

func TestAuthenticatedUser_UnknownSession(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	authUser, err := svc.AuthenticatedUser(context.Background(), "unknown-session")
	if err == nil {
		t.Fatal("Expected error for unknown session")
	}

	var notFoundErr *domain.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("Expected NotFoundError, got %T: %v", err, err)
	}

	if authUser != nil {
		t.Errorf("Expected nil user, got %v", authUser)
	}
}

func TestAuthenticatedUser_DeletedSession(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	sessionID, _, validationErr, err := svc.Signin(context.Background(), user.Username(), "test")
	if err != nil {
		t.Fatalf("Signin returned unexpected error: %v", err)
	}
	if validationErr != nil {
		t.Fatalf("Signin returned unexpected validation error: %v", validationErr)
	}

	if signoutErr := svc.Signout(context.Background(), sessionID); signoutErr != nil {
		t.Fatalf("Signout returned unexpected error: %v", signoutErr)
	}

	if _, authErr := svc.AuthenticatedUser(context.Background(), sessionID); authErr == nil {
		t.Fatal("Expected error for signed-out session")
	}
}

func TestSignout_DeletesSession(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	sessionID, expiresAt, validationErr, err := svc.Signin(context.Background(), user.Username(), "test")
	if err != nil {
		t.Fatalf("Signin returned unexpected error: %v", err)
	}
	if validationErr != nil {
		t.Fatalf("Signin returned unexpected validation error: %v", validationErr)
	}
	_ = expiresAt

	if signoutErr := svc.Signout(context.Background(), sessionID); signoutErr != nil {
		t.Fatalf("Signout returned unexpected error: %v", signoutErr)
	}

	_, sessionErr := s.GetSession(context.Background(), sessionID)
	if sessionErr == nil {
		t.Fatal("Expected session to be deleted")
	}
}
