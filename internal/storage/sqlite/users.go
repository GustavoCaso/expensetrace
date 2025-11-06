package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func (s *sqliteStorage) CreateUser(ctx context.Context, username, passwordHash string) (storage.User, error) {
	// Begin transaction to ensure atomic user + exclude category creation
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Will be no-op if committed
	}()

	// Create user
	createdAt := time.Now()
	result, err := tx.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, created_at) VALUES (?, ?, ?)`,
		username, passwordHash, createdAt.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user id: %w", err)
	}

	// Create Exclude category for this user
	_, err = tx.ExecContext(ctx,
		`INSERT INTO categories (name, pattern, user_id) VALUES (?, ?, ?)`,
		storage.ExcludeCategory, "$a", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create exclude category: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return storage.NewUser(userID, username, passwordHash, createdAt), nil
}

func (s *sqliteStorage) GetUserByUsername(ctx context.Context, username string) (storage.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at
		FROM users
		WHERE username = ?
	`, username)

	var id int64
	var uname string
	var passwordHash string
	var createdAt int64

	err := row.Scan(&id, &uname, &passwordHash, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &storage.NotFoundError{}
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	return storage.NewUser(id, uname, passwordHash, time.Unix(createdAt, 0)), nil
}

func (s *sqliteStorage) GetUserByID(ctx context.Context, id int64) (storage.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at
		FROM users
		WHERE id = ?
	`, id)

	var userID int64
	var username string
	var passwordHash string
	var createdAt int64

	err := row.Scan(&userID, &username, &passwordHash, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &storage.NotFoundError{}
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	return storage.NewUser(userID, username, passwordHash, time.Unix(createdAt, 0)), nil
}

func (s *sqliteStorage) UpdateUsername(ctx context.Context, userID int64, newUsername string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET username = ?
		WHERE id = ?
	`, newUsername, userID)
	if err != nil {
		return fmt.Errorf("failed to update username: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return &storage.NotFoundError{}
	}

	return nil
}

func (s *sqliteStorage) UpdatePassword(ctx context.Context, userID int64, newPasswordHash string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET password_hash = ?
		WHERE id = ?
	`, newPasswordHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return &storage.NotFoundError{}
	}

	return nil
}
