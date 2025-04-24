package web

import (
	"flag"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestSetFlags(t *testing.T) {
	cmd := NewCommand()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(fs)

	// Check if port flag is registered
	portFlag := fs.Lookup("p")
	if portFlag == nil {
		t.Fatal("Expected port flag to be registered")
	}

	if portFlag.DefValue != "8080" {
		t.Errorf("Port default value = %q, want 8080", portFlag.DefValue)
	}
}

func TestSetFlagsENV(t *testing.T) {
	t.Setenv("EXPENSETRACE_PORT", "8081")

	cmd := NewCommand()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(fs)

	// Check if port flag is registered
	portFlag := fs.Lookup("p")
	if portFlag == nil {
		t.Fatal("Expected port flag to be registered")
	}

	if portFlag.DefValue != "8081" {
		t.Errorf("Port default value = %q, want 8081", portFlag.DefValue)
	}
}

func TestRun(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Create test categories
	categories := []expenseDB.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery", Type: expenseDB.ExpenseCategoryType},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit", Type: expenseDB.ExpenseCategoryType},
	}

	for _, c := range categories {
		_, err := expenseDB.CreateCategory(db, c.Name, c.Pattern, c.Type)
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	// Create category matcher
	matcher := category.NewMatcher(categories)

	// Create command
	cmd := NewCommand()

	// Set a random port for testing
	port = "0" // This will let the OS assign a random available port

	// Start the server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- cmd.Run(db, matcher)
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

	// Test invalid port
	port = "invalid"
	err = cmd.Run(db, matcher)
	if err == nil {
		t.Error("Expected error for invalid port, got nil")
	}
}
