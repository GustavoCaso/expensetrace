package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mattn/go-sqlite3"

	"github.com/GustavoCaso/expensetrace/domain"
)

func (s *sqliteStorage) CreateAPIToken(
	ctx context.Context,
	userID int64,
	name, tokenHash string,
) (domain.APIToken, error) {
	createdAt := time.Now()
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO api_tokens (user_id, name, token_hash, created_at)
		VALUES (?, ?, ?, ?)
	`, userID, name, tokenHash, createdAt.Unix())
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return nil, &domain.AlreadyExistsError{}
		}
		return nil, fmt.Errorf("failed to create api token: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get api token id: %w", err)
	}

	return domain.NewAPIToken(id, userID, name, createdAt, nil), nil
}

func (s *sqliteStorage) GetAPITokenByHash(ctx context.Context, tokenHash string) (domain.APIToken, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, created_at, last_used_at
		FROM api_tokens
		WHERE token_hash = ?
	`, tokenHash)

	token, err := scanAPIToken(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{}
		}
		return nil, fmt.Errorf("failed to scan api token: %w", err)
	}

	return token, nil
}

func (s *sqliteStorage) GetAPITokens(ctx context.Context, userID int64) ([]domain.APIToken, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, created_at, last_used_at
		FROM api_tokens
		WHERE user_id = ?
		ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query api tokens: %w", err)
	}
	defer rows.Close()

	tokens := []domain.APIToken{}
	for rows.Next() {
		token, scanErr := scanAPIToken(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan api token: %w", scanErr)
		}
		tokens = append(tokens, token)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate api tokens: %w", err)
	}

	return tokens, nil
}

func (s *sqliteStorage) DeleteAPIToken(ctx context.Context, userID, id int64) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM api_tokens WHERE id = ? AND user_id = ?
	`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete api token: %w", err)
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

func (s *sqliteStorage) TouchAPIToken(ctx context.Context, id int64, usedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE api_tokens SET last_used_at = ? WHERE id = ?
	`, usedAt.Unix(), id)
	if err != nil {
		return fmt.Errorf("failed to update api token last_used_at: %w", err)
	}

	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAPIToken(row rowScanner) (domain.APIToken, error) {
	var id, userID, createdAt int64
	var name string
	var lastUsedAt sql.NullInt64

	if err := row.Scan(&id, &userID, &name, &createdAt, &lastUsedAt); err != nil {
		return nil, err
	}

	var lastUsedAtPtr *time.Time
	if lastUsedAt.Valid {
		t := time.Unix(lastUsedAt.Int64, 0)
		lastUsedAtPtr = &t
	}

	return domain.NewAPIToken(id, userID, name, time.Unix(createdAt, 0), lastUsedAtPtr), nil
}
