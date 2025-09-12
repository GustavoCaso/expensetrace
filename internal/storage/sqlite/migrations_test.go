package sqlite

import (
	"context"
	"testing"
)

func TestMigrations(t *testing.T) {
	stor := setupTestStorage(t)

	// Test that storage was created successfully with migrations applied
	// This verifies that all tables exist and are properly structured
	_, err := stor.GetExpenses(context.Background())
	if err != nil {
		t.Fatalf("Failed to query expenses table after migrations: %v", err)
	}

	_, err = stor.GetCategories(context.Background())
	if err != nil {
		t.Fatalf("Failed to query categories table after migrations: %v", err)
	}

	// Test that we can create data (which verifies schema)
	_, err = stor.CreateCategory(context.Background(), "Test Category", "test.*")
	if err != nil {
		t.Fatalf("Failed to create category after migrations: %v", err)
	}
}
