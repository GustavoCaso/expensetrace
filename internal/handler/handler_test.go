package handler

import (
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestNew(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
	}
	matcher := category.NewMatcher(categories)

	// Create server
	handler := New(database, matcher, logger)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}
