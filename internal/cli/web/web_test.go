package web

import (
	"database/sql"
	"flag"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	err = expenseDB.CreateExpenseTable(database)
	if err != nil {
		t.Fatalf("Failed to create expenses table: %v", err)
	}

	err = expenseDB.CreateCategoriesTable(database)
	if err != nil {
		t.Fatalf("Failed to create categories table: %v", err)
	}

	return database
}

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()
	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	webCmd, ok := cmd.(webCommand)
	if !ok {
		t.Fatal("Expected webCommand type")
	}

	if desc := webCmd.Description(); desc != "Web interface" {
		t.Errorf("Description = %q, want Web interface", desc)
	}
}

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

func TestRun(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

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
	port = "0" // This will let the OS assign a random available port

	// Start the server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- cmd.Run(db, matcher)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test that the server is responding
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
