package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
)

func (s *sqliteStorage) CreatePasswordResetToken(
	ctx context.Context,
	userID int64,
	token string,
	expiresAt time.Time,
) (domain.PasswordResetToken, error) {
	createdAt := time.Now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO password_reset_tokens (token, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, token, userID, expiresAt.Unix(), createdAt.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to create password reset token: %w", err)
	}

	return domain.NewPasswordResetToken(token, userID, expiresAt, nil, createdAt), nil
}

func (s *sqliteStorage) GetPasswordResetToken(ctx context.Context, token string) (domain.PasswordResetToken, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT token, user_id, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token = ?
	`, token)

	var tok string
	var userID int64
	var expiresAt int64
	var usedAt sql.NullInt64
	var createdAt int64

	err := row.Scan(&tok, &userID, &expiresAt, &usedAt, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &domain.NotFoundError{}
		}
		return nil, fmt.Errorf("failed to scan password reset token: %w", err)
	}

	var usedAtPtr *time.Time
	if usedAt.Valid {
		t := time.Unix(usedAt.Int64, 0)
		usedAtPtr = &t
	}

	return domain.NewPasswordResetToken(tok, userID, time.Unix(expiresAt, 0), usedAtPtr, time.Unix(createdAt, 0)), nil
}

func (s *sqliteStorage) MarkPasswordResetTokenUsed(ctx context.Context, token string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE password_reset_tokens
		SET used_at = ?
		WHERE token = ? AND used_at IS NULL
	`, time.Now().Unix(), token)
	if err != nil {
		return fmt.Errorf("failed to mark password reset token used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return &domain.NotFoundError{}
	}

	return nil
}

func (s *sqliteStorage) InvalidatePasswordResetTokensForUser(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE password_reset_tokens
		SET used_at = ?
		WHERE user_id = ? AND used_at IS NULL
	`, time.Now().Unix(), userID)
	if err != nil {
		return fmt.Errorf("failed to invalidate password reset tokens for user: %w", err)
	}

	return nil
}
