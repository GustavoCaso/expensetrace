package profile

import (
	"context"
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestUpdateUsername_RejectsDuplicate(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	// Create a second user whose username we'll try to steal.
	_, createErr := s.CreateUser(context.Background(), "otheruser", "hash")
	if createErr != nil {
		t.Fatalf("Failed to create other user: %v", createErr)
	}

	svc := New(s, logger)

	validationErr, err := svc.UpdateUsername(context.Background(), user.ID(), "otheruser")
	if err != nil {
		t.Fatalf("UpdateUsername returned unexpected internal error: %v", err)
	}

	if validationErr == nil {
		t.Fatal("Expected validationErr for duplicate username")
	}

	const expectedMsg = "username already exists"
	if validationErr.Error() != expectedMsg {
		t.Fatalf("Expected validationErr message %q, got %q", expectedMsg, validationErr.Error())
	}

	// Original username unchanged.
	refreshed, getErr := s.GetUserByID(context.Background(), user.ID())
	if getErr != nil {
		t.Fatalf("Failed to get user: %v", getErr)
	}
	if refreshed.Username() != user.Username() {
		t.Fatalf("Expected username to remain %q, got %q", user.Username(), refreshed.Username())
	}
}

func TestUpdatePassword_RejectsWrongCurrentPassword(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	validationErr, err := svc.UpdatePassword(
		context.Background(),
		user.ID(),
		"wrongpassword",
		"newpassword123",
		"newpassword123",
	)
	if err != nil {
		t.Fatalf("UpdatePassword returned unexpected internal error: %v", err)
	}

	if validationErr == nil {
		t.Fatal("Expected validationErr for wrong current password")
	}

	const expectedMsg = "current password is incorrect"
	if validationErr.Error() != expectedMsg {
		t.Fatalf("Expected validationErr message %q, got %q", expectedMsg, validationErr.Error())
	}
}

func TestUpdatePassword_RejectsShortPassword(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	validationErr, err := svc.UpdatePassword(context.Background(), user.ID(), "test", "short", "short")
	if err != nil {
		t.Fatalf("UpdatePassword returned unexpected internal error: %v", err)
	}

	if validationErr == nil {
		t.Fatal("Expected validationErr for short password")
	}

	const expectedMsg = "password must be at least 8 characters long"
	if validationErr.Error() != expectedMsg {
		t.Fatalf("Expected validationErr message %q, got %q", expectedMsg, validationErr.Error())
	}
}
