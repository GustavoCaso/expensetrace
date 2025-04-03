package testutil

import (
	"database/sql"
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	err = db.ApplyMigrations(database)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Enable foreign key constraints
	_, err = database.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable PRAGMA: %v", err)
	}

	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	})

	return database
}
