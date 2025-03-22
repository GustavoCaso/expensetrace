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

	err = db.CreateExpenseTable(database)
	if err != nil {
		t.Fatalf("Failed to create expense table: %v", err)
	}

	err = db.CreateCategoriesTable(database)
	if err != nil {
		t.Fatalf("Failed to create categories table: %v", err)
	}

	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("Failed to close test database: %v", err)
		}
	})

	return database
}
