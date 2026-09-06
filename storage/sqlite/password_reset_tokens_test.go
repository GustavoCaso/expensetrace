package sqlite_test

import (
	"errors"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/testutil"
)

func TestCreateAndGetPasswordResetToken(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	expiresAt := time.Now().Add(time.Hour).Truncate(time.Second)

	created, err := s.CreatePasswordResetToken(t.Context(), user.ID(), "sometoken", expiresAt)
	if err != nil {
		t.Fatalf("CreatePasswordResetToken returned unexpected error: %v", err)
	}

	if created.Token() != "sometoken" {
		t.Errorf("Expected token 'sometoken', got %q", created.Token())
	}
	if created.UserID() != user.ID() {
		t.Errorf("Expected user id %d, got %d", user.ID(), created.UserID())
	}
	if !created.ExpiresAt().Equal(expiresAt) {
		t.Errorf("Expected expiresAt %v, got %v", expiresAt, created.ExpiresAt())
	}
	if created.UsedAt() != nil {
		t.Errorf("Expected UsedAt to be nil for freshly created token")
	}

	fetched, err := s.GetPasswordResetToken(t.Context(), "sometoken")
	if err != nil {
		t.Fatalf("GetPasswordResetToken returned unexpected error: %v", err)
	}

	if fetched.UserID() != user.ID() {
		t.Errorf("Expected fetched user id %d, got %d", user.ID(), fetched.UserID())
	}
}

func TestMarkPasswordResetTokenUsed_PreservesOriginalUsedAt(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	expiresAt := time.Now().Add(time.Hour).Truncate(time.Second)

	if _, err := s.CreatePasswordResetToken(t.Context(), user.ID(), "sometoken", expiresAt); err != nil {
		t.Fatalf("CreatePasswordResetToken returned unexpected error: %v", err)
	}

	if err := s.MarkPasswordResetTokenUsed(t.Context(), "sometoken"); err != nil {
		t.Fatalf("first MarkPasswordResetTokenUsed returned unexpected error: %v", err)
	}

	firstUsed, err := s.GetPasswordResetToken(t.Context(), "sometoken")
	if err != nil {
		t.Fatalf("GetPasswordResetToken returned unexpected error: %v", err)
	}
	if firstUsed.UsedAt() == nil {
		t.Fatal("Expected token to be marked used")
	}
	originalUsedAt := *firstUsed.UsedAt()

	err = s.MarkPasswordResetTokenUsed(t.Context(), "sometoken")
	if err == nil {
		t.Fatal("Expected error when marking an already-used token used again")
	}
	var notFoundErr *domain.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("Expected NotFoundError, got %T: %v", err, err)
	}

	secondUsed, err := s.GetPasswordResetToken(t.Context(), "sometoken")
	if err != nil {
		t.Fatalf("GetPasswordResetToken returned unexpected error: %v", err)
	}
	if secondUsed.UsedAt() == nil || !secondUsed.UsedAt().Equal(originalUsedAt) {
		t.Errorf("Expected original used_at %v to be preserved, got %v", originalUsedAt, secondUsed.UsedAt())
	}
}

func TestInvalidatePasswordResetTokensForUser(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	expiresAt := time.Now().Add(time.Hour).Truncate(time.Second)

	if _, err := s.CreatePasswordResetToken(t.Context(), user.ID(), "token-a", expiresAt); err != nil {
		t.Fatalf("CreatePasswordResetToken (a) returned unexpected error: %v", err)
	}
	if _, err := s.CreatePasswordResetToken(t.Context(), user.ID(), "token-b", expiresAt); err != nil {
		t.Fatalf("CreatePasswordResetToken (b) returned unexpected error: %v", err)
	}

	if err := s.MarkPasswordResetTokenUsed(t.Context(), "token-b"); err != nil {
		t.Fatalf("MarkPasswordResetTokenUsed returned unexpected error: %v", err)
	}

	if err := s.InvalidatePasswordResetTokensForUser(t.Context(), user.ID()); err != nil {
		t.Fatalf("InvalidatePasswordResetTokensForUser returned unexpected error: %v", err)
	}

	tokenA, err := s.GetPasswordResetToken(t.Context(), "token-a")
	if err != nil {
		t.Fatalf("GetPasswordResetToken (a) returned unexpected error: %v", err)
	}
	if tokenA.UsedAt() == nil {
		t.Error("Expected previously-unused token to be marked used after invalidation")
	}

	tokenB, err := s.GetPasswordResetToken(t.Context(), "token-b")
	if err != nil {
		t.Fatalf("GetPasswordResetToken (b) returned unexpected error: %v", err)
	}
	if tokenB.UsedAt() == nil {
		t.Error("Expected already-used token to remain marked used")
	}
}

func TestGetPasswordResetToken_NotFound(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	_, err := s.GetPasswordResetToken(t.Context(), "does-not-exist")
	if err == nil {
		t.Fatal("Expected error for unknown token")
	}

	var notFoundErr *domain.NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("Expected NotFoundError, got %T: %v", err, err)
	}
}
