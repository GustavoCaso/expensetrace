package auth

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/util"
)

const (
	// resetTokenBytes controls the length of the generated reset token (in
	// bytes of randomness, resulting in twice as many hex characters).
	resetTokenBytes = 16
	// resetTokenDuration is how long a password reset token is valid for.
	resetTokenDuration = time.Hour
)

// RequestPasswordReset generates a password reset token for the given
// username and logs it. If the username is unknown, it returns an empty
// token and no error, to avoid leaking which usernames exist.
func (s *Service) RequestPasswordReset(ctx context.Context, username string) (string, time.Time, error) {
	user, getErr := s.storage.GetUserByUsername(ctx, username)
	if getErr != nil {
		var notFoundErr *domain.NotFoundError
		if errors.As(getErr, &notFoundErr) {
			return "", time.Time{}, nil
		}
		s.logger.Error("Failed to get user for password reset", "error", getErr)
		return "", time.Time{}, getErr
	}

	token := util.GenerateRandomID(resetTokenBytes)
	expiresAt := time.Now().Add(resetTokenDuration)

	if _, createErr := s.storage.CreatePasswordResetToken(ctx, user.ID(), token, expiresAt); createErr != nil {
		s.logger.Error("Failed to create password reset token", "error", createErr)
		return "", time.Time{}, createErr
	}

	s.logger.Warn("Password reset requested",
		"username", user.Username(),
		"token", token,
		"expires_at", expiresAt)

	return token, expiresAt, nil
}

// ResetPassword validates the reset token and new password, updates the
// user's password, marks the token used, and invalidates existing sessions.
// validationErr holds a user-facing message meant to be re-displayed on the
// reset password page; err holds an unexpected/internal failure.
func (s *Service) ResetPassword(
	ctx context.Context,
	token, newPassword, confirmPassword string,
) (error, error) {
	resetToken, getErr := s.storage.GetPasswordResetToken(ctx, token)
	if getErr != nil {
		var notFoundErr *domain.NotFoundError
		if errors.As(getErr, &notFoundErr) {
			//nolint:staticcheck // user-facing validation message
			return errors.New("Invalid or expired reset link"), nil
		}
		s.logger.Error("Failed to get password reset token", "error", getErr)
		return nil, getErr
	}

	if resetToken.UsedAt() != nil || time.Now().After(resetToken.ExpiresAt()) {
		//nolint:staticcheck // user-facing validation message
		return errors.New("Invalid or expired reset link"), nil
	}

	if newPassword == "" {
		//nolint:staticcheck // user-facing validation message
		return errors.New("Password is required"), nil
	}

	if newPassword != confirmPassword {
		//nolint:staticcheck // user-facing validation message
		return errors.New("Passwords do not match"), nil
	}

	if len(newPassword) < minPasswordLength {
		//nolint:staticcheck // user-facing validation message
		return errors.New("Password must be at least 8 characters long"), nil
	}

	hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if hashErr != nil {
		s.logger.Error("Failed to hash password", "error", hashErr)
		return nil, hashErr
	}

	if updateErr := s.storage.UpdatePassword(ctx, resetToken.UserID(), string(hashedPassword)); updateErr != nil {
		s.logger.Error("Failed to update password", "error", updateErr)
		return nil, updateErr
	}

	if markErr := s.storage.MarkPasswordResetTokenUsed(ctx, token); markErr != nil {
		s.logger.Error("Failed to mark password reset token used", "error", markErr)
		return nil, markErr
	}

	if invalidateErr := s.storage.InvalidatePasswordResetTokensForUser(ctx, resetToken.UserID()); invalidateErr != nil {
		s.logger.Error("Failed to invalidate outstanding password reset tokens", "error", invalidateErr)
		return nil, invalidateErr
	}

	if sessionErr := s.storage.DeleteSessionsForUser(ctx, resetToken.UserID()); sessionErr != nil {
		s.logger.Error("Failed to invalidate sessions after password reset", "error", sessionErr)
		return nil, sessionErr
	}

	return nil, nil //nolint:nilnil // (validationErr, err) pair; both nil means success
}
