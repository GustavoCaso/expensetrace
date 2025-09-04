package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestRun(t *testing.T) {
	logger := testutil.TestLogger(t)
	db := testutil.SetupTestDB(t, logger)

	// Create test categories
	categories := []expenseDB.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
	}

	for _, c := range categories {
		_, err := expenseDB.CreateCategory(db, c.Name, c.Pattern)
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	// Create category matcher
	matcher := category.NewMatcher(categories)

	// Create command
	cmd := NewCommand()

	// Set a random port for testing
	t.Setenv("EXPENSETRACE_PORT", "0") // This will let the OS assign a random available port

	// Start the server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- cmd.Run(db, matcher, logger)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test that the server is responding
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make a test request
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}
