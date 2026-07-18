// Package profile contains the business logic for updating a user's
// username and password, independent of the HTTP layer.
package profile

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/logger"
	"github.com/GustavoCaso/expensetrace/storage"
)

// minPasswordLength is the minimum accepted password length.
const minPasswordLength = 8

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

// UpdateUsername validates and updates a user's username. The original
// handler always rendered the same profile page with an error message
// regardless of whether the failure was a validation issue or an unexpected
// storage error (it never returned a distinct HTTP status for internal
// failures), so all failure paths here are represented as validationErr to
// preserve that exact behavior. err is reserved for future use but is not
// currently produced by any path.
func (s *Service) UpdateUsername(
	ctx context.Context,
	userID int64,
	newUsername string,
) (error, error) {
	if newUsername == "" {
		return errors.New("username is required"), nil
	}

	// Check if username already exists
	_, getErr := s.storage.GetUserByUsername(ctx, newUsername)
	if getErr == nil {
		return errors.New("username already exists"), nil
	}

	var notFoundErr *domain.NotFoundError
	if !errors.As(getErr, &notFoundErr) {
		s.logger.Error("Failed to check username", "error", getErr)
		return errors.New("internal Server Error"), nil
	}

	if updateErr := s.storage.UpdateUsername(ctx, userID, newUsername); updateErr != nil {
		s.logger.Error("Failed to update username", "error", updateErr, "user_id", userID)
		return errors.New("failed to update username"), nil
	}

	s.logger.Info("Username updated", "user_id", userID, "new_username", newUsername)
	return nil, nil //nolint:nilnil // both return values are error; nil,nil means success
}

// UpdatePassword validates and updates a user's password. As with
// UpdateUsername, the original handler always rendered the same profile
// page with an error message for every failure path (never a distinct HTTP
// status), so all failures are represented as validationErr here.
func (s *Service) UpdatePassword(
	ctx context.Context,
	userID int64,
	currentPassword, newPassword, confirmPassword string,
) (error, error) {
	if currentPassword == "" || newPassword == "" || confirmPassword == "" {
		return errors.New("all fields are required"), nil
	}

	if newPassword != confirmPassword {
		return errors.New("new passwords do not match"), nil
	}

	if len(newPassword) < minPasswordLength {
		return errors.New("password must be at least 8 characters long"), nil
	}

	user, getErr := s.storage.GetUserByID(ctx, userID)
	if getErr != nil {
		s.logger.Error("Failed to get user", "error", getErr, "user_id", userID)
		return errors.New("internal Server Error"), nil
	}

	if compareErr := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash()), []byte(currentPassword),
	); compareErr != nil {
		return errors.New("current password is incorrect"), nil
	}

	hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if hashErr != nil {
		s.logger.Error("Failed to hash password", "error", hashErr)
		return errors.New("internal Server Error"), nil
	}

	if updateErr := s.storage.UpdatePassword(ctx, userID, string(hashedPassword)); updateErr != nil {
		s.logger.Error("Failed to update password", "error", updateErr, "user_id", userID)
		return errors.New("failed to update password"), nil
	}

	s.logger.Info("Password updated", "user_id", userID)
	return nil, nil //nolint:nilnil // both return values are error; nil,nil means success
}
