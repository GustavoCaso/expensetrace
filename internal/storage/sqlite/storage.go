package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	// import sqlite driver.
	_ "github.com/mattn/go-sqlite3"

	"github.com/GustavoCaso/expensetrace/internal/config"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

type sqliteStorage struct {
	db *sql.DB
}

func New(dbConfig config.DBConfig) (storage.Storage, error) {
	db, err := sql.Open("sqlite3", dbConfig.Source)
	if err != nil {
		return nil, err
	}

	// Apply connection pool settings
	if dbConfig.MaxOpenConns > 0 {
		db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	}

	if dbConfig.MaxIdleConns > 0 {
		db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	}

	if dbConfig.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(dbConfig.ConnMaxLifetime)
	}

	if dbConfig.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(dbConfig.ConnMaxIdleTime)
	}

	ctx := context.Background()

	// Enable foreign key constraints
	_, err = db.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	if err != nil {
		return nil, err
	}

	// Apply SQLite PRAGMA settings
	if dbConfig.JournalMode != "" {
		_, err = db.ExecContext(ctx, fmt.Sprintf("PRAGMA journal_mode = %s", dbConfig.JournalMode))
		if err != nil {
			return nil, fmt.Errorf("failed to set journal_mode: %w", err)
		}
	}

	if dbConfig.Synchronous != "" {
		_, err = db.ExecContext(ctx, fmt.Sprintf("PRAGMA synchronous = %s", dbConfig.Synchronous))
		if err != nil {
			return nil, fmt.Errorf("failed to set synchronous: %w", err)
		}
	}

	if dbConfig.CacheSize != 0 {
		_, err = db.ExecContext(ctx, fmt.Sprintf("PRAGMA cache_size = %d", dbConfig.CacheSize))
		if err != nil {
			return nil, fmt.Errorf("failed to set cache_size: %w", err)
		}
	}

	if dbConfig.BusyTimeout > 0 {
		_, err = db.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout = %d", dbConfig.BusyTimeout))
		if err != nil {
			return nil, fmt.Errorf("failed to set busy_timeout: %w", err)
		}
	}

	if dbConfig.WALAutocheckpoint > 0 {
		_, err = db.ExecContext(ctx, fmt.Sprintf("PRAGMA wal_autocheckpoint = %d", dbConfig.WALAutocheckpoint))
		if err != nil {
			return nil, fmt.Errorf("failed to set wal_autocheckpoint: %w", err)
		}
	}

	if dbConfig.TempStore != "" {
		_, err = db.ExecContext(ctx, fmt.Sprintf("PRAGMA temp_store = %s", dbConfig.TempStore))
		if err != nil {
			return nil, fmt.Errorf("failed to set temp_store: %w", err)
		}
	}

	return &sqliteStorage{db: db}, nil
}

func (s *sqliteStorage) Close() error {
	return s.db.Close()
}
