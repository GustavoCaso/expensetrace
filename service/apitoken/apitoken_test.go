package apitoken

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/testutil"
)

func TestCreateAndAuthenticate(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	svc := New(s, logger)
	ctx := t.Context()

	token, plaintext, validationErr, err := svc.Create(ctx, user.ID(), "  my agent  ")
	if err != nil || validationErr != nil {
		t.Fatalf("Create: err=%v validationErr=%v", err, validationErr)
	}
	if token.Name() != "my agent" {
		t.Errorf("expected trimmed name, got %q", token.Name())
	}
	if !strings.HasPrefix(plaintext, TokenPrefix) {
		t.Errorf("token should start with %q: %q", TokenPrefix, plaintext)
	}

	// The plaintext must never be persisted.
	stored, err := s.GetAPITokenByHash(ctx, plaintext)
	if err == nil && stored != nil {
		t.Fatal("plaintext token should not be usable as a stored hash")
	}

	authed, err := svc.Authenticate(ctx, plaintext)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if authed.UserID() != user.ID() {
		t.Errorf("expected user %d, got %d", user.ID(), authed.UserID())
	}

	tokens, err := svc.List(ctx, user.ID())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tokens) != 1 || tokens[0].LastUsedAt() == nil {
		t.Fatalf("expected one token with last_used_at recorded, got %+v", tokens)
	}
}

func TestCreateValidation(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	svc := New(s, logger)

	for name, input := range map[string]string{
		"empty":    "",
		"blank":    "   ",
		"too long": strings.Repeat("a", maxNameLength+1),
	} {
		t.Run(name, func(t *testing.T) {
			_, _, validationErr, err := svc.Create(t.Context(), user.ID(), input)
			if err != nil {
				t.Fatalf("unexpected internal error: %v", err)
			}
			if validationErr == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestCreateRejectsDuplicateName(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	svc := New(s, logger)

	if _, _, validationErr, err := svc.Create(t.Context(), user.ID(), "agent"); err != nil || validationErr != nil {
		t.Fatalf("Create: %v %v", err, validationErr)
	}

	_, plaintext, validationErr, err := svc.Create(t.Context(), user.ID(), " agent ")
	if err != nil {
		t.Fatalf("duplicate name should not be an internal error: %v", err)
	}
	if validationErr == nil || !strings.Contains(validationErr.Error(), "already exists") {
		t.Errorf("expected already-exists validation error, got %v", validationErr)
	}
	if plaintext != "" {
		t.Error("no token should be returned for a duplicate name")
	}
}

func TestAuthenticateRejectsUnknownAndRevoked(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	svc := New(s, logger)
	ctx := t.Context()

	for _, bad := range []string{"", "nope", TokenPrefix + "deadbeef"} {
		if _, err := svc.Authenticate(ctx, bad); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("Authenticate(%q): expected ErrInvalidToken, got %v", bad, err)
		}
	}

	token, plaintext, _, err := svc.Create(ctx, user.ID(), "temp")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err = svc.Revoke(ctx, user.ID(), token.ID()); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err = svc.Authenticate(ctx, plaintext); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("revoked token: expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthenticateThrottlesLastUsedWrites(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	svc := New(s, logger)
	ctx := t.Context()

	now := time.Unix(1_700_000_000, 0)
	svc.now = func() time.Time { return now }

	_, plaintext, _, err := svc.Create(ctx, user.ID(), "agent")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	lastUsed := func() time.Time {
		t.Helper()
		tokens, listErr := svc.List(ctx, user.ID())
		if listErr != nil || len(tokens) != 1 || tokens[0].LastUsedAt() == nil {
			t.Fatalf("List: %v %+v", listErr, tokens)
		}
		return *tokens[0].LastUsedAt()
	}

	if _, err = svc.Authenticate(ctx, plaintext); err != nil {
		t.Fatal(err)
	}
	first := lastUsed()

	now = now.Add(touchInterval / 2)
	if _, err = svc.Authenticate(ctx, plaintext); err != nil {
		t.Fatal(err)
	}
	if !lastUsed().Equal(first) {
		t.Error("last_used_at should not change within the touch interval")
	}

	now = now.Add(touchInterval)
	if _, err = svc.Authenticate(ctx, plaintext); err != nil {
		t.Fatal(err)
	}
	if !lastUsed().After(first) {
		t.Error("last_used_at should advance after the touch interval")
	}
}
