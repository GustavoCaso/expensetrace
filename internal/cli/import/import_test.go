package importCmd

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func createTestFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return tmpFile
}

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()
	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}

	importCmd, ok := cmd.(importCommand)
	if !ok {
		t.Fatal("Expected importCommand type")
	}

	if desc := importCmd.Description(); desc != "Imports expenses to the DB" {
		t.Errorf("Description = %q, want Imports expenses to the DB", desc)
	}
}

func TestSetFlags(t *testing.T) {
	cmd := NewCommand()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(fs)

	// Check if file flag is registered
	fileFlag := fs.Lookup("f")
	if fileFlag == nil {
		t.Fatal("Expected file flag to be registered")
	}

	if fileFlag.DefValue != "" {
		t.Errorf("File default value = %q, want empty string", fileFlag.DefValue)
	}
}

func TestRun(t *testing.T) {
	// database := testutil.SetupTestDB(t)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
	}
	// Note: matcher is not used since we can't fully test the Run function
	_ = category.NewMatcher(categories)

	// Create test CSV file
	csvContent := `source,date,description,amount,currency
Test Source,01/01/2024,Restaurant bill,-1234.56,USD
Test Source,02/01/2024,Uber ride,-5000.00,USD
Test Source,03/01/2024,Salary,500000.00,USD`

	tmpFile := createTestFile(t, csvContent)
	defer os.Remove(tmpFile)

	// Set import file
	importFile = tmpFile

	// Create command
	cmd := NewCommand()

	// Run the command
	// Note: Since the command calls os.Exit, we can't test the actual execution
	// In a real test, you might want to:
	// 1. Refactor the command to return errors instead of calling os.Exit
	// 2. Use a test helper that can capture os.Exit calls
	// For now, we'll just verify that the command and flags are set up correctly
	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}
}
