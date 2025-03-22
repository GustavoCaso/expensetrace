package router

import (
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
)

func TestNew(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
	}
	matcher := category.NewMatcher(categories)

	// Create router
	handler := New(database, matcher)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}
