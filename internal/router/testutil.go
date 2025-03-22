package router

import (
	"database/sql"
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	err = db.CreateExpenseTable(database)
	if err != nil {
		t.Fatalf("Failed to create expenses table: %v", err)
	}

	err = db.CreateCategoriesTable(database)
	if err != nil {
		t.Fatalf("Failed to create categories table: %v", err)
	}

	return database
}

// newTestRouter creates a router instance for testing
func newTestRouter(db *sql.DB, matcher *category.Matcher) *router {
	return &router{
		reload:  false,
		matcher: matcher,
		db:      db,
	}
}
