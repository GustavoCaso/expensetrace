// Package auth contains the business logic for user signup, signin and
// signout, independent of the HTTP layer.
package auth

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

const (
	// SessionDuration is how long a session is valid for after creation.
	SessionDuration = 7 * 24 * time.Hour // 7 days
	// sessionIDBytes controls the length of the generated session ID (in
	// bytes of randomness, resulting in twice as many hex characters).
	sessionIDBytes = 16 // Results in 32 hex characters
	// minPasswordLength is the minimum accepted password length.
	minPasswordLength = 8
)

type Service struct {
	storage storage.Storage
	logger  *logger.Logger
}

func New(storage storage.Storage, logger *logger.Logger) *Service {
	return &Service{
		storage: storage,
		logger:  logger,
	}
}

// Signup validates the signup form, creates the user, an initial session and
// the default Exclude category for the user. validationErr holds a
// user-facing message meant to be re-displayed on the signup page; err holds
// an unexpected/internal failure that should result in a 500.
func (s *Service) Signup(
	ctx context.Context,
	username, password, confirmPassword string,
) (string, time.Time, error, error) {
	if username == "" || password == "" {
		//nolint:staticcheck // user-facing validation message
		return "", time.Time{}, errors.New("Username and password are required"), nil
	}

	if password != confirmPassword {
		//nolint:staticcheck // user-facing validation message
		return "", time.Time{}, errors.New("Passwords do not match"), nil
	}

	if len(password) < minPasswordLength {
		//nolint:staticcheck // user-facing validation message
		return "", time.Time{}, errors.New("Password must be at least 8 characters long"), nil
	}

	hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if hashErr != nil {
		s.logger.Error("Failed to hash password", "error", hashErr)
		return "", time.Time{}, nil, hashErr
	}

	user, createErr := s.storage.CreateUser(ctx, username, string(hashedPassword))
	if createErr != nil {
		s.logger.Error("Failed to create user", "error", createErr, "username", username)
		//nolint:staticcheck // user-facing validation message
		return "", time.Time{}, errors.New("Username already exists or database error occurred"), nil
	}

	sessionID := util.GenerateRandomID(sessionIDBytes)
	expiresAt := time.Now().Add(SessionDuration)
	_, sessionErr := s.storage.CreateSession(ctx, user.ID(), sessionID, expiresAt)
	if sessionErr != nil {
		s.logger.Error("Failed to create session", "error", sessionErr)
		return "", time.Time{}, nil, sessionErr
	}

	// Create default Exclude category for new user
	_, excludeErr := s.storage.CreateCategory(ctx, user.ID(), storage.ExcludeCategory, "$a", 0)
	if excludeErr != nil {
		s.logger.Error("Failed to create exclude category for new user", "error", excludeErr, "user_id", user.ID())
	}

	return sessionID, expiresAt, nil, nil
}

// Signin validates credentials and creates a new session. validationErr
// holds a user-facing message meant to be re-displayed on the signin page;
// err holds an unexpected/internal failure that should result in a 500.
func (s *Service) Signin(
	ctx context.Context,
	username, password string,
) (string, time.Time, error, error) {
	if username == "" || password == "" {
		//nolint:staticcheck // user-facing validation message
		return "", time.Time{}, errors.New("Username and password are required"), nil
	}

	user, getErr := s.storage.GetUserByUsername(ctx, username)
	if getErr != nil {
		var notFoundErr *storage.NotFoundError
		if errors.As(getErr, &notFoundErr) {
			//nolint:staticcheck // user-facing validation message
			return "", time.Time{}, errors.New("Invalid username or password"), nil
		}
		s.logger.Error("Failed to get user", "error", getErr)
		return "", time.Time{}, nil, getErr
	}

	if compareErr := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash()), []byte(password),
	); compareErr != nil {
		//nolint:staticcheck // user-facing validation message
		return "", time.Time{}, errors.New("Invalid username or password"), nil
	}

	sessionID := util.GenerateRandomID(sessionIDBytes)
	expiresAt := time.Now().Add(SessionDuration)
	_, sessionErr := s.storage.CreateSession(ctx, user.ID(), sessionID, expiresAt)
	if sessionErr != nil {
		s.logger.Error("Failed to create session", "error", sessionErr)
		return "", time.Time{}, nil, sessionErr
	}

	return sessionID, expiresAt, nil, nil
}

// Signout deletes the given session.
func (s *Service) Signout(ctx context.Context, sessionID string) error {
	return s.storage.DeleteSession(ctx, sessionID)
}
