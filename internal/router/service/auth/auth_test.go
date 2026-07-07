package auth

import (
	"context"
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/storage"
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
		if c.Name() == storage.ExcludeCategory {
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
