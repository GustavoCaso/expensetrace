package sqlite

import (
	"errors"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
)

func TestAPITokenLifecycle(t *testing.T) {
	s, user := setupTestStorage(t)
	ctx := t.Context()

	created, err := s.CreateAPIToken(ctx, user.ID(), "laptop", "hash-1")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}
	if created.ID() == 0 || created.Name() != "laptop" || created.UserID() != user.ID() {
		t.Fatalf("unexpected created token: id=%d name=%q user=%d", created.ID(), created.Name(), created.UserID())
	}
	if created.LastUsedAt() != nil {
		t.Error("new token should have no last_used_at")
	}

	got, err := s.GetAPITokenByHash(ctx, "hash-1")
	if err != nil {
		t.Fatalf("GetAPITokenByHash: %v", err)
	}
	if got.ID() != created.ID() {
		t.Errorf("expected id %d, got %d", created.ID(), got.ID())
	}

	usedAt := time.Unix(1_700_000_000, 0)
	if err = s.TouchAPIToken(ctx, created.ID(), usedAt); err != nil {
		t.Fatalf("TouchAPIToken: %v", err)
	}
	got, err = s.GetAPITokenByHash(ctx, "hash-1")
	if err != nil {
		t.Fatalf("GetAPITokenByHash after touch: %v", err)
	}
	if got.LastUsedAt() == nil || !got.LastUsedAt().Equal(usedAt) {
		t.Errorf("expected last_used_at %v, got %v", usedAt, got.LastUsedAt())
	}

	if _, err = s.CreateAPIToken(ctx, user.ID(), "server", "hash-2"); err != nil {
		t.Fatalf("CreateAPIToken second: %v", err)
	}

	tokens, err := s.GetAPITokens(ctx, user.ID())
	if err != nil {
		t.Fatalf("GetAPITokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}

	if err = s.DeleteAPIToken(ctx, user.ID(), created.ID()); err != nil {
		t.Fatalf("DeleteAPIToken: %v", err)
	}

	_, err = s.GetAPITokenByHash(ctx, "hash-1")
	var notFound *domain.NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected NotFoundError after delete, got %v", err)
	}
}

func TestAPITokenHashIsUnique(t *testing.T) {
	s, user := setupTestStorage(t)

	if _, err := s.CreateAPIToken(t.Context(), user.ID(), "a", "same-hash"); err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}
	if _, err := s.CreateAPIToken(t.Context(), user.ID(), "b", "same-hash"); err == nil {
		t.Error("expected duplicate token hash to fail")
	}
}

func TestDeleteAPITokenIsScopedToUser(t *testing.T) {
	s, user := setupTestStorage(t)
	ctx := t.Context()

	other, err := s.CreateUser(ctx, "other", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token, err := s.CreateAPIToken(ctx, user.ID(), "mine", "hash-mine")
	if err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	err = s.DeleteAPIToken(ctx, other.ID(), token.ID())
	var notFound *domain.NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("expected NotFoundError deleting another user's token, got %v", err)
	}

	otherTokens, err := s.GetAPITokens(ctx, other.ID())
	if err != nil {
		t.Fatalf("GetAPITokens: %v", err)
	}
	if len(otherTokens) != 0 {
		t.Errorf("other user should see no tokens, got %d", len(otherTokens))
	}

	if _, err = s.GetAPITokenByHash(ctx, "hash-mine"); err != nil {
		t.Errorf("token should still exist: %v", err)
	}
}

func TestAPITokenNameIsUniquePerUser(t *testing.T) {
	s, user := setupTestStorage(t)
	ctx := t.Context()

	if _, err := s.CreateAPIToken(ctx, user.ID(), "laptop", "hash-1"); err != nil {
		t.Fatalf("CreateAPIToken: %v", err)
	}

	_, err := s.CreateAPIToken(ctx, user.ID(), "laptop", "hash-2")
	var exists *domain.AlreadyExistsError
	if !errors.As(err, &exists) {
		t.Fatalf("expected AlreadyExistsError for duplicate name, got %v", err)
	}

	other, err := s.CreateUser(ctx, "other", "hash")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err = s.CreateAPIToken(ctx, other.ID(), "laptop", "hash-3"); err != nil {
		t.Errorf("another user should be able to reuse the name: %v", err)
	}
}
