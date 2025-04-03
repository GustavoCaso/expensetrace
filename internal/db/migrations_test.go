package db

import (
	"testing"
	"time"
)

func TestDropDB(t *testing.T) {
	database := setupTestDB(t)

	// Insert test expense
	_, err := database.Exec("INSERT INTO expenses(source, amount, description, expense_type, date, currency, category_id) VALUES(?, ?, ?, ?, ?, ?, ?)",
		"test", 1000, "Test expense", ChargeType, time.Now().Unix(), "USD", nil)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	// Insert test category
	_, err = database.Exec("INSERT INTO categories(name, pattern) VALUES(?, ?)",
		"test", "*")
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	err = DropTables(database)
	if err != nil {
		t.Errorf("Failed to delete expenses table: %v", err)
	}

	// Verify tables were deleted
	_, err = database.Query("SELECT * FROM expenses")
	if err == nil {
		t.Error("Expected error when querying expenses table, got nil")
	}

	_, err = database.Query("SELECT * FROM categories")
	if err == nil {
		t.Error("Expected error when querying category table, got nil")
	}

	_, err = database.Query("SELECT * FROM schema_migrations")
	if err == nil {
		t.Error("Expected error when querying schema_migrations table, got nil")
	}
}
