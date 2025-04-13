package importutil

import (
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestImportCSV(t *testing.T) {
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

	// Test CSV data
	csvData := `Test Source,01/01/2024,Restaurant bill,-1234.56,USD
Test Source,02/01/2024,Uber ride,-5000.00,USD
Test Source,03/01/2024,Salary,500000.00,USD`

	reader := strings.NewReader(csvData)
	errs := Import("test.csv", reader, database, matcher)
	if len(errs) > 0 {
		t.Errorf("Import failed with errors: %v", errs)
	}

	// Verify imported expenses
	expenses, err := db.GetExpenses(database)
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(expenses) != 3 {
		t.Fatalf("Expected 3 expenses, got %d", len(expenses))
	}

	// Verify first expense
	if expenses[0].Source != "Test Source" {
		t.Errorf("Expense[0].Source = %v, want Test Source", expenses[0].Source)
	}
	if expenses[0].Description != "restaurant bill" {
		t.Errorf("Expense[0].Description = %v, want restaurant bill", expenses[0].Description)
	}
	if expenses[0].Amount != -123456 {
		t.Errorf("Expense[0].Amount = %v, want -123456", expenses[0].Amount)
	}
	if expenses[0].Type != db.ChargeType {
		t.Errorf("Expense[0].Type = %v, want ChargeType", expenses[0].Type)
	}
	if expenses[0].Currency != "USD" {
		t.Errorf("Expense[0].Currency = %v, want USD", expenses[0].Currency)
	}
	if expenses[0].CategoryID.Int64 != 1 {
		t.Errorf("Expense[0].CategoryID = %v, want 1", expenses[0].CategoryID)
	}

	// Verify second expense
	if expenses[1].Description != "uber ride" {
		t.Errorf("Expense[1].Description = %v, want uber ride", expenses[1].Description)
	}
	if expenses[1].Amount != -500000 {
		t.Errorf("Expense[1].Amount = %v, want -500000", expenses[1].Amount)
	}
	if expenses[1].CategoryID.Int64 != 2 {
		t.Errorf("Expense[1].CategoryID = %v, want 2", expenses[1].CategoryID)
	}

	// Verify third expense
	if expenses[2].Amount != 50000000 {
		t.Errorf("Expense[2].Amount = %v, want 50000000", expenses[2].Amount)
	}
	if expenses[2].Type != db.IncomeType {
		t.Errorf("Expense[2].Type = %v, want IncomeType", expenses[2].Type)
	}
}

func TestImportJSON(t *testing.T) {
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

	// Test JSON data
	jsonData := `[
		{
			"source": "Test Source",
			"date": "2024-01-01T00:00:00Z",
			"description": "Restaurant bill",
			"amount": -123456,
			"currency": "USD"
		},
		{
			"source": "Test Source",
			"date": "2024-01-02T00:00:00Z",
			"description": "Uber ride",
			"amount": -500000,
			"currency": "USD"
		},
		{
			"source": "Test Source",
			"date": "2024-01-03T00:00:00Z",
			"description": "Salary",
			"amount": 50000000,
			"currency": "USD"
		}
	]`

	reader := strings.NewReader(jsonData)
	errs := Import("test.json", reader, database, matcher)
	if len(errs) > 0 {
		t.Errorf("Import failed with errors: %v", errs)
	}

	// Verify imported expenses
	expenses, err := db.GetExpenses(database)
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(expenses) != 3 {
		t.Fatalf("Expected 3 expenses, got %d", len(expenses))
	}

	// Verify first expense
	if expenses[0].Source != "Test Source" {
		t.Errorf("Expense[0].Source = %v, want Test Source", expenses[0].Source)
	}
	if expenses[0].Description != "restaurant bill" {
		t.Errorf("Expense[0].Description = %v, want restaurant bill", expenses[0].Description)
	}
	if expenses[0].Amount != -123456 {
		t.Errorf("Expense[0].Amount = %v, want -123456", expenses[0].Amount)
	}
	if expenses[0].Type != db.ChargeType {
		t.Errorf("Expense[0].Type = %v, want ChargeType", expenses[0].Type)
	}
	if expenses[0].Currency != "USD" {
		t.Errorf("Expense[0].Currency = %v, want USD", expenses[0].Currency)
	}
	if expenses[0].CategoryID.Int64 != 1 {
		t.Errorf("Expense[0].CategoryID = %v, want 1", expenses[0].CategoryID)
	}

	// Verify second expense
	if expenses[1].Description != "uber ride" {
		t.Errorf("Expense[1].Description = %v, want uber ride", expenses[1].Description)
	}
	if expenses[1].Amount != -500000 {
		t.Errorf("Expense[1].Amount = %v, want -500000", expenses[1].Amount)
	}
	if expenses[1].CategoryID.Int64 != 2 {
		t.Errorf("Expense[1].CategoryID = %v, want 2", expenses[1].CategoryID)
	}

	// Verify third expense
	if expenses[2].Amount != 50000000 {
		t.Errorf("Expense[2].Amount = %v, want 50000000", expenses[2].Amount)
	}
	if expenses[2].Type != db.IncomeType {
		t.Errorf("Expense[2].Type = %v, want IncomeType", expenses[2].Type)
	}
}

func TestImportInvalidFormat(t *testing.T) {
	database := testutil.SetupTestDB(t)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
	}
	matcher := category.NewMatcher(categories)

	// Test with invalid file format
	reader := strings.NewReader("test data")
	errs := Import("test.txt", reader, database, matcher)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errs))
	}
	if !strings.Contains(errs[0].Error(), "unsupported file format") {
		t.Errorf("Expected error about unsupported format, got %v", errs[0])
	}
}

func TestImportInvalidCSV(t *testing.T) {
	database := testutil.SetupTestDB(t)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
	}
	matcher := category.NewMatcher(categories)

	// Test with invalid CSV data
	csvData := `Test Source,invalid-date,Restaurant bill,-1234.56,USD`

	reader := strings.NewReader(csvData)
	errs := Import("test.csv", reader, database, matcher)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errs))
	}
}

func TestImportInvalidJSON(t *testing.T) {
	database := testutil.SetupTestDB(t)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
	}
	matcher := category.NewMatcher(categories)

	// Test with invalid JSON data
	jsonData := `[
		{
			"source": "Test Source",
			"date": "invalid-date",
			"description": "Restaurant bill",
			"amount": -123456,
			"currency": "USD"
		}
	]`

	reader := strings.NewReader(jsonData)
	errs := Import("test.json", reader, database, matcher)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errs))
	}
}
