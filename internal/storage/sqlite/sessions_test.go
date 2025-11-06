package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func TestCreateSession(t *testing.T) {
	s, _ := setupTestStorage(t)

	// Create a user first
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user, err := s.CreateUser(context.Background(), "testuser", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create session
	sessionID := "test-session-id-123"
	expiresAt := time.Now().Add(24 * time.Hour)
	session, err := s.CreateSession(context.Background(), user.ID(), sessionID, expiresAt)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.ID() != sessionID {
		t.Errorf("Expected session ID '%s', got '%s'", sessionID, session.ID())
	}

	if session.UserID() != user.ID() {
		t.Errorf("Expected user ID %d, got %d", user.ID(), session.UserID())
	}

	if session.CreatedAt().IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestGetSession(t *testing.T) {
	s, _ := setupTestStorage(t)

	// Create a user first
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user, err := s.CreateUser(context.Background(), "testuser", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create session
	sessionID := "test-session-id-456"
	expiresAt := time.Now().Add(24 * time.Hour)
	created, err := s.CreateSession(context.Background(), user.ID(), sessionID, expiresAt)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get session
	session, err := s.GetSession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if session.ID() != created.ID() {
		t.Errorf("Expected session ID '%s', got '%s'", created.ID(), session.ID())
	}

	if session.UserID() != user.ID() {
		t.Errorf("Expected user ID %d, got %d", user.ID(), session.UserID())
	}
}

func TestGetSessionExpired(t *testing.T) {
	s, _ := setupTestStorage(t)

	// Create a user first
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user, err := s.CreateUser(context.Background(), "testuser", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create session that's already expired
	sessionID := "expired-session-id"
	expiresAt := time.Now().Add(-1 * time.Hour) // Expired 1 hour ago
	_, err = s.CreateSession(context.Background(), user.ID(), sessionID, expiresAt)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Try to get expired session
	_, err = s.GetSession(context.Background(), sessionID)
	if err == nil {
		t.Error("Expected error when getting expired session")
	}

	var notFoundErr *storage.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("Expected NotFoundError for expired session, got %v", err)
	}
}

func TestGetSessionNotFound(t *testing.T) {
	s, _ := setupTestStorage(t)

	// Try to get non-existent session
	_, err := s.GetSession(context.Background(), "nonexistent-session")
	if err == nil {
		t.Error("Expected error when getting non-existent session")
	}

	var notFoundErr *storage.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("Expected NotFoundError, got %v", err)
	}
}

func TestDeleteSession(t *testing.T) {
	s, _ := setupTestStorage(t)
	// Create a user first
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user, err := s.CreateUser(context.Background(), "testuser", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create session
	sessionID := "session-to-delete"
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = s.CreateSession(context.Background(), user.ID(), sessionID, expiresAt)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Delete session
	err = s.DeleteSession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Try to get deleted session
	_, err = s.GetSession(context.Background(), sessionID)
	if err == nil {
		t.Error("Expected error when getting deleted session")
	}
}

func TestDeleteExpiredSessions(t *testing.T) {
	s, _ := setupTestStorage(t)

	// Create a user first
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	user, err := s.CreateUser(context.Background(), "testuser", string(hashedPassword))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create expired session
	expiredSessionID := "expired-session"
	_, err = s.CreateSession(context.Background(), user.ID(), expiredSessionID, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("Failed to create expired session: %v", err)
	}

	// Create valid session
	validSessionID := "valid-session"
	_, err = s.CreateSession(context.Background(), user.ID(), validSessionID, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Failed to create valid session: %v", err)
	}

	// Delete expired sessions
	err = s.DeleteExpiredSessions(context.Background())
	if err != nil {
		t.Fatalf("Failed to delete expired sessions: %v", err)
	}

	// Verify expired session is gone
	_, err = s.GetSession(context.Background(), expiredSessionID)
	if err == nil {
		t.Error("Expired session should have been deleted")
	}

	// Verify valid session still exists
	_, err = s.GetSession(context.Background(), validSessionID)
	if err != nil {
		t.Error("Valid session should not have been deleted")
	}
}
