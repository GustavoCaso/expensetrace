package router

import (
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestNew(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestStorage(t, logger)

	// Create test categories
	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
		storage.NewCategory(2, "Transport", "uber|taxi|transit"),
	}
	matcher := matcher.New(categories)

	// Create router
	handler, _ := New(database, matcher, logger)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}
