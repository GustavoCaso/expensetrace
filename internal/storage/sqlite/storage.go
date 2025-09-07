package sqlite

import (
	"context"
	"database/sql"

	// import sqlite driver.
	_ "github.com/mattn/go-sqlite3"

	"github.com/GustavoCaso/expensetrace/internal/storage"
)

type sqliteStorage struct {
	db *sql.DB
}

func New(dbsource string) (storage.Storage, error) {
	db, err := sql.Open("sqlite3", dbsource)
	if err != nil {
		return nil, err
	}

	// Enable foreign key constraints
	_, err = db.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
	if err != nil {
		return nil, err
	}

	return &sqliteStorage{db: db}, nil
}

func (s *sqliteStorage) Close() error {
	return s.db.Close()
}
