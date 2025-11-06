package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func (s *sqliteStorage) CreateSession(
	ctx context.Context,
	userID int64,
	sessionID string,
	expiresAt time.Time,
) (storage.Session, error) {
	statement, err := s.db.PrepareContext(ctx, `
		INSERT INTO sessions (id, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare create session statement: %w", err)
	}
	defer statement.Close()

	createdAt := time.Now()
	_, err = statement.ExecContext(ctx, sessionID, userID, expiresAt.Unix(), createdAt.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return storage.NewSession(sessionID, userID, expiresAt, createdAt), nil
}

func (s *sqliteStorage) GetSession(ctx context.Context, sessionID string) (storage.Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at, created_at
		FROM sessions
		WHERE id = ? AND expires_at > ?
	`, sessionID, time.Now().Unix())

	var id string
	var userID int64
	var expiresAt int64
	var createdAt int64

	err := row.Scan(&id, &userID, &expiresAt, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &storage.NotFoundError{}
		}
		return nil, fmt.Errorf("failed to scan session: %w", err)
	}

	return storage.NewSession(id, userID, time.Unix(expiresAt, 0), time.Unix(createdAt, 0)), nil
}

func (s *sqliteStorage) DeleteSession(ctx context.Context, sessionID string) error {
	statement, err := s.db.PrepareContext(ctx, `
		DELETE FROM sessions WHERE id = ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare delete session statement: %w", err)
	}
	defer statement.Close()

	_, err = statement.ExecContext(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (s *sqliteStorage) DeleteExpiredSessions(ctx context.Context) error {
	statement, err := s.db.PrepareContext(ctx, `
		DELETE FROM sessions WHERE expires_at <= ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare delete expired sessions statement: %w", err)
	}
	defer statement.Close()

	_, err = statement.ExecContext(ctx, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return nil
}
