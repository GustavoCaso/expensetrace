package sqlite

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func TestCreateUser(t *testing.T) {
	s, _ := setupTestStorage(t)

	// Hash a password
	password := "testpassword123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Create user
	user, err := s.CreateUser(context.Background(), "testuser", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID() == 0 {
		t.Error("Expected user ID to be set")
	}

	if user.Username() != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username())
	}

	if user.PasswordHash() != string(hashedPassword) {
		t.Error("Password hash mismatch")
	}

	if user.CreatedAt().IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestCreateUserDuplicate(t *testing.T) {
	s, _ := setupTestStorage(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)

	// Create first user
	_, err := s.CreateUser(context.Background(), "testuser", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create first user: %v", err)
	}

	// Try to create duplicate user
	_, err = s.CreateUser(context.Background(), "testuser", string(hashedPassword))
	if err == nil {
		t.Error("Expected error when creating duplicate user")
	}
}

func TestGetUserByUsername(t *testing.T) {
	s, _ := setupTestStorage(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)

	// Create user
	created, err := s.CreateUser(context.Background(), "testuser", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Get user by username
	user, err := s.GetUserByUsername(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("Failed to get user by username: %v", err)
	}

	if user.ID() != created.ID() {
		t.Errorf("Expected user ID %d, got %d", created.ID(), user.ID())
	}

	if user.Username() != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username())
	}
}

func TestGetUserByUsernameNotFound(t *testing.T) {
	s, _ := setupTestStorage(t)

	// Try to get non-existent user
	_, err := s.GetUserByUsername(context.Background(), "nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent user")
	}

	var notFoundErr *storage.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("Expected NotFoundError, got %v", err)
	}
}

func TestGetUserByID(t *testing.T) {
	s, _ := setupTestStorage(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)

	// Create user
	created, err := s.CreateUser(context.Background(), "testuser", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Get user by ID
	user, err := s.GetUserByID(context.Background(), created.ID())
	if err != nil {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	if user.Username() != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username())
	}
}

func TestPasswordVerification(t *testing.T) {
	s, _ := setupTestStorage(t)

	password := "mysecretpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// Create user
	user, err := s.CreateUser(context.Background(), "testuser", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify correct password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash()), []byte(password))
	if err != nil {
		t.Error("Password verification failed for correct password")
	}

	// Verify incorrect password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash()), []byte("wrongpassword"))
	if err == nil {
		t.Error("Password verification should fail for incorrect password")
	}
}

func TestUpdateUsername(t *testing.T) {
	s, _ := setupTestStorage(t)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)

	// Create user
	user, err := s.CreateUser(context.Background(), "oldusername", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Update username
	err = s.UpdateUsername(context.Background(), user.ID(), "newusername")
	if err != nil {
		t.Fatalf("Failed to update username: %v", err)
	}

	// Verify update
	updated, err := s.GetUserByID(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}

	if updated.Username() != "newusername" {
		t.Errorf("Expected username 'newusername', got '%s'", updated.Username())
	}

	// Verify old username doesn't work
	_, err = s.GetUserByUsername(context.Background(), "oldusername")
	var notFoundErr *storage.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Error("Expected NotFoundError when getting user with old username")
	}

	// Verify new username works
	byUsername, err := s.GetUserByUsername(context.Background(), "newusername")
	if err != nil {
		t.Fatalf("Failed to get user by new username: %v", err)
	}
	if byUsername.ID() != user.ID() {
		t.Error("User ID mismatch after username update")
	}
}

func TestUpdateUsernameNotFound(t *testing.T) {
	s, _ := setupTestStorage(t)

	// Try to update non-existent user
	err := s.UpdateUsername(context.Background(), 99999, "newusername")
	if err == nil {
		t.Error("Expected error when updating non-existent user")
	}

	var notFoundErr *storage.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("Expected NotFoundError, got %v", err)
	}
}

func TestUpdatePassword(t *testing.T) {
	s, _ := setupTestStorage(t)

	oldPassword := "oldpassword123"
	oldHashedPassword, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)

	// Create user
	user, err := s.CreateUser(context.Background(), "testuser", string(oldHashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Update password
	newPassword := "newpassword456"
	newHashedPassword, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)

	err = s.UpdatePassword(context.Background(), user.ID(), string(newHashedPassword))
	if err != nil {
		t.Fatalf("Failed to update password: %v", err)
	}

	// Verify update
	updated, err := s.GetUserByID(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}

	// Verify old password doesn't work
	err = bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash()), []byte(oldPassword))
	if err == nil {
		t.Error("Old password should not work after update")
	}

	// Verify new password works
	err = bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash()), []byte(newPassword))
	if err != nil {
		t.Error("New password verification failed")
	}
}

func TestUpdatePasswordNotFound(t *testing.T) {
	s, _ := setupTestStorage(t)

	newHashedPassword, _ := bcrypt.GenerateFromPassword([]byte("newpassword"), bcrypt.DefaultCost)

	// Try to update non-existent user
	err := s.UpdatePassword(context.Background(), 99999, string(newHashedPassword))
	if err == nil {
		t.Error("Expected error when updating password for non-existent user")
	}

	var notFoundErr *storage.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("Expected NotFoundError, got %v", err)
	}
}
