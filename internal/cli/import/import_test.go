package importCmd

import (
	"database/sql"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
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
	database := testutil.SetupTestDB(t)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
	}

	for _, c := range categories {
		_, err := db.CreateCategory(database, c.Name, c.Pattern)
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	matcher := category.NewMatcher(categories)

	// Create test CSV file with various amount formats
	csvContent := `Test Source,01/01/2024,Restaurant bill,-1234.56,USD
Test Source,02/01/2024,Uber ride,-5000.00,USD
Test Source,03/01/2024,Salary,500000.00,USD
Test Source,04/01/2024,Small charge,-10.50,USD
Test Source,05/01/2024,Small income,10.50,USD`

	tmpFile := createTestFile(t, csvContent)
	defer os.Remove(tmpFile)

	// Test case 1: Missing file flag
	importFile = ""
	cmd := NewCommand()
	err := cmd.Run(database, matcher)
	if err == nil || err.Error() != "you must provide a file to import" {
		t.Errorf("Expected error about missing file, got: %v", err)
	}

	// Test case 2: Non-existent file
	importFile = "nonexistent.csv"
	err = cmd.Run(database, matcher)
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("Expected file not found error, got: %v", err)
	}

	// Test case 3: Successful import
	importFile = tmpFile
	err = cmd.Run(database, matcher)
	if err != nil {
		t.Fatalf("Failed to import file: %v", err)
	}

	// Verify imported data
	rows, err := database.Query(`
		SELECT date, description, amount, currency, category_id
		FROM expenses
		ORDER BY date
	`)
	if err != nil {
		t.Fatalf("Failed to query expenses: %v", err)
	}
	defer rows.Close()

	expectedExpenses := []struct {
		date       string
		desc       string
		amount     float64
		currency   string
		categoryID int64
	}{
		{"2024-01-01", "restaurant bill", -123456, "USD", 1},
		{"2024-01-02", "uber ride", -500000, "USD", 2},
		{"2024-01-03", "salary", 50000000, "USD", 0},
		{"2024-01-04", "small charge", -1050, "USD", 0},
		{"2024-01-05", "small income", 1050, "USD", 0},
	}

	for i, expected := range expectedExpenses {
		if !rows.Next() {
			t.Fatalf("Expected %d rows, got fewer", len(expectedExpenses))
		}

		var timestamp int64
		var desc, currency string
		var amount float64
		var categoryID sql.NullInt64
		err := rows.Scan(&timestamp, &desc, &amount, &currency, &categoryID)
		if err != nil {
			t.Fatalf("Failed to scan row %d: %v", i+1, err)
		}

		// Convert timestamp to date string for comparison
		date := time.Unix(timestamp, 0).Format("2006-01-02")
		if date != expected.date {
			t.Errorf("Row %d: date = %q, want %q", i+1, date, expected.date)
		}
		if desc != strings.ToLower(expected.desc) {
			t.Errorf("Row %d: description = %q, want %q", i+1, desc, expected.desc)
		}
		if amount != expected.amount {
			t.Errorf("Row %d: amount = %v, want %v", i+1, amount, expected.amount)
		}
		if currency != expected.currency {
			t.Errorf("Row %d: currency = %q, want %q", i+1, currency, expected.currency)
		}
		if categoryID.Valid && categoryID.Int64 != expected.categoryID {
			t.Errorf("Row %d: category_id = %v, want %v", i+1, categoryID.Int64, expected.categoryID)
		}
	}

	if rows.Next() {
		t.Error("Got more rows than expected")
	}
	if err = rows.Err(); err != nil {
		t.Errorf("Error iterating rows: %v", err)
	}
}
