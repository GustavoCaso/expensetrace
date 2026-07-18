package router

import (
	"testing"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/testutil"
)

func TestNew(t *testing.T) {
	logger := testutil.TestLogger(t)
	database, _ := testutil.SetupTestStorage(t, logger)

	// Create router
	handler := New(database, logger)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}

func TestCategoryMatcherIncludesExcludeCategory(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory(t.Context(), user.ID(), "Entertainment", "cinema|movie", 0)
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	r := newRouter(s, logger)

	m, err := r.categoryMatcher(t.Context(), user.ID())
	if err != nil {
		t.Fatalf("categoryMatcher returned error: %v", err)
	}

	foundExclude := false
	for _, c := range m.Categories() {
		if c.Name() == domain.ExcludeCategory {
			foundExclude = true
		}
	}

	if !foundExclude {
		t.Fatal("Expected matcher categories to include the exclude category")
	}

	matchedID, matchedName := m.Match("cinema")
	if matchedID == nil || *matchedID != categoryID {
		t.Fatalf("Expected 'cinema' to match category %d, got %v", categoryID, matchedID)
	}

	if matchedName != "Entertainment" {
		t.Fatalf("Expected matched category name 'Entertainment', got %s", matchedName)
	}
}
