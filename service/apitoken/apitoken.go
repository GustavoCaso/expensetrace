// Package apitoken manages long-lived bearer tokens that let external clients,
// such as AI agents connecting over MCP, read a user's data.
//
// Tokens are random, shown to the user once at creation, and stored only as a
// SHA-256 hash. Unlike passwords they carry 256 bits of entropy, so a fast
// hash is sufficient and keeps per-request verification cheap.
package apitoken

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/logger"
	"github.com/GustavoCaso/expensetrace/storage"
	"github.com/GustavoCaso/expensetrace/util"
)

const (
	// TokenPrefix marks a string as an ExpenseTrace API token, which makes
	// leaked tokens easy to recognise and grep for.
	TokenPrefix = "et_"
	// tokenBytes is the amount of randomness in a token.
	tokenBytes = 32
	// maxNameLength bounds the user-provided token label.
	maxNameLength = 100
	// touchInterval limits how often last_used_at is written, so a busy
	// agent does not cause a database write on every request.
	touchInterval = time.Minute
)

// ErrInvalidToken is returned by Authenticate when the token is unknown.
var ErrInvalidToken = errors.New("invalid api token")

type Service struct {
	storage storage.Storage
	logger  *logger.Logger
	now     func() time.Time
}

func New(storage storage.Storage, logger *logger.Logger) *Service {
	return &Service{
		storage: storage,
		logger:  logger,
		now:     time.Now,
	}
}

// Create generates a new token for the user. It returns the stored token
// metadata and the plaintext token, which is not recoverable afterwards.
// validationErr holds a user-facing message; err an internal failure.
func (s *Service) Create(
	ctx context.Context,
	userID int64,
	name string,
) (domain.APIToken, string, error, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		//nolint:staticcheck // user-facing validation message
		return nil, "", errors.New("Token name is required"), nil
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		//nolint:staticcheck // user-facing validation message
		return nil, "", errors.New("Token name must be 100 characters or fewer"), nil
	}

	plaintext := TokenPrefix + util.GenerateRandomID(tokenBytes)

	token, err := s.storage.CreateAPIToken(ctx, userID, name, hashToken(plaintext))
	if err != nil {
		// The user already has a token with this name, e.g. because the
		// browser resubmitted the create form on refresh.
		var existsErr *domain.AlreadyExistsError
		if errors.As(err, &existsErr) {
			//nolint:staticcheck // user-facing validation message
			return nil, "", fmt.Errorf("A token named %q already exists", name), nil
		}
		s.logger.Error("Failed to create api token", "error", err, "user_id", userID)
		return nil, "", nil, err
	}

	s.logger.Info("API token created", "user_id", userID, "token_id", token.ID(), "name", name)

	return token, plaintext, nil, nil
}

// List returns the user's tokens, newest first.
func (s *Service) List(ctx context.Context, userID int64) ([]domain.APIToken, error) {
	return s.storage.GetAPITokens(ctx, userID)
}

// Revoke deletes one of the user's tokens.
func (s *Service) Revoke(ctx context.Context, userID, id int64) error {
	if err := s.storage.DeleteAPIToken(ctx, userID, id); err != nil {
		return err
	}

	s.logger.Info("API token revoked", "user_id", userID, "token_id", id)

	return nil
}

// Authenticate resolves a plaintext token to its stored record. It returns
// ErrInvalidToken when the token does not exist.
func (s *Service) Authenticate(ctx context.Context, plaintext string) (domain.APIToken, error) {
	if !strings.HasPrefix(plaintext, TokenPrefix) {
		return nil, ErrInvalidToken
	}

	token, err := s.storage.GetAPITokenByHash(ctx, hashToken(plaintext))
	if err != nil {
		var notFoundErr *domain.NotFoundError
		if errors.As(err, &notFoundErr) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	now := s.now()
	if last := token.LastUsedAt(); last == nil || now.Sub(*last) >= touchInterval {
		if touchErr := s.storage.TouchAPIToken(ctx, token.ID(), now); touchErr != nil {
			// Not fatal: the token is valid, we just failed to record usage.
			s.logger.Warn("Failed to record api token usage", "error", touchErr, "token_id", token.ID())
		}
	}

	return token, nil
}

func hashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
