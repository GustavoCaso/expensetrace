package importutil

import (
	"context"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestImportCSV(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	// Create test categories
	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
		storage.NewCategory(2, "Transport", "uber|taxi|transit"),
	}

	for _, c := range categories {
		_, err := s.CreateCategory(context.Background(), c.Name(), c.Pattern())
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}
	matcher := matcher.New(categories)

	// Test CSV data
	csvData := `Test Source,01/01/2024,Restaurant bill,-1234.56,USD
Test Source,02/01/2024,Uber ride,-5000.00,USD
Test Source,03/01/2024,Salary,500000.00,USD`

	reader := strings.NewReader(csvData)
	info := Import(context.Background(), "test.csv", reader, s, matcher)
	if info.Error != nil {
		t.Errorf("Import failed with error: %v", info.Error)
	}

	// Verify imported expenses
	expenses, err := s.GetAllExpenseTypes(context.Background())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(expenses) != 3 {
		t.Fatalf("Expected 3 expenses, got %d", len(expenses))
	}

	// Verify first expense
	if expenses[0].Source() != "Test Source" {
		t.Errorf("Expense[0].Source = %v, want Test Source", expenses[0].Source())
	}
	if expenses[0].Description() != "restaurant bill" {
		t.Errorf("Expense[0].Description = %v, want restaurant bill", expenses[0].Description())
	}
	if expenses[0].Amount() != -123456 {
		t.Errorf("Expense[0].Amount = %v, want -123456", expenses[0].Amount())
	}
	if expenses[0].Type() != storage.ChargeType {
		t.Errorf("Expense[0].Type = %v, want ChargeType", expenses[0].Type())
	}
	if expenses[0].Currency() != "USD" {
		t.Errorf("Expense[0].Currency = %v, want USD", expenses[0].Currency())
	}
	if expenses[0].CategoryID() == nil && *expenses[0].CategoryID() == 1 {
		t.Errorf("Expense[0].CategoryID = %v, want 1", expenses[0].CategoryID())
	}

	// Verify second expense
	if expenses[1].Description() != "uber ride" {
		t.Errorf("Expense[1].Description = %v, want uber ride", expenses[1].Description())
	}
	if expenses[1].Amount() != -500000 {
		t.Errorf("Expense[1].Amount = %v, want -500000", expenses[1].Amount())
	}
	if expenses[1].CategoryID() == nil && *expenses[0].CategoryID() == 2 {
		t.Errorf("Expense[1].CategoryID = %v, want 2", expenses[1].CategoryID())
	}

	// Verify third expense
	if expenses[2].Amount() != 50000000 {
		t.Errorf("Expense[2].Amount = %v, want 50000000", expenses[2].Amount())
	}
	if expenses[2].Type() != storage.IncomeType {
		t.Errorf("Expense[2].Type = %v, want IncomeType", expenses[2].Type())
	}
}

func TestImportJSON(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	// Create test categories
	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
		storage.NewCategory(2, "Transport", "uber|taxi|transit"),
	}

	for _, c := range categories {
		_, err := s.CreateCategory(context.Background(), c.Name(), c.Pattern())
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	matcher := matcher.New(categories)

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
	info := Import(context.Background(), "test.json", reader, s, matcher)
	if info.Error != nil {
		t.Errorf("Import failed with error: %v", info.Error)
	}

	// Verify imported expenses
	expenses, err := s.GetAllExpenseTypes(context.Background())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(expenses) != 3 {
		t.Fatalf("Expected 3 expenses, got %d", len(expenses))
	}

	// Verify first expense
	if expenses[0].Source() != "Test Source" {
		t.Errorf("Expense[0].Source = %v, want Test Source", expenses[0].Source())
	}
	if expenses[0].Description() != "restaurant bill" {
		t.Errorf("Expense[0].Description = %v, want restaurant bill", expenses[0].Description())
	}
	if expenses[0].Amount() != -123456 {
		t.Errorf("Expense[0].Amount = %v, want -123456", expenses[0].Amount())
	}
	if expenses[0].Type() != storage.ChargeType {
		t.Errorf("Expense[0].Type = %v, want ChargeType", expenses[0].Type())
	}
	if expenses[0].Currency() != "USD" {
		t.Errorf("Expense[0].Currency = %v, want USD", expenses[0].Currency())
	}
	if expenses[0].CategoryID() == nil && *expenses[0].CategoryID() == 1 {
		t.Errorf("Expense[0].CategoryID = %v, want 1", expenses[0].CategoryID())
	}

	// Verify second expense
	if expenses[1].Description() != "uber ride" {
		t.Errorf("Expense[1].Description = %v, want uber ride", expenses[1].Description())
	}
	if expenses[1].Amount() != -500000 {
		t.Errorf("Expense[1].Amount = %v, want -500000", expenses[1].Amount())
	}
	if expenses[1].CategoryID() == nil && *expenses[0].CategoryID() == 2 {
		t.Errorf("Expense[1].CategoryID = %v, want 2", expenses[1].CategoryID())
	}

	// Verify third expense
	if expenses[2].Amount() != 50000000 {
		t.Errorf("Expense[2].Amount = %v, want 50000000", expenses[2].Amount())
	}
	if expenses[2].Type() != storage.IncomeType {
		t.Errorf("Expense[2].Type = %v, want IncomeType", expenses[2].Type())
	}
}

func TestImportInvalidFormat(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	// Create test categories
	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
		storage.NewCategory(2, "Transport", "uber|taxi|transit"),
	}

	matcher := matcher.New(categories)

	// Test with invalid file format
	reader := strings.NewReader("test data")
	info := Import(context.Background(), "test.txt", reader, s, matcher)
	if info.Error == nil || info.Error.Error() != "unsupported file format: .txt" {
		t.Errorf("Expected error for unsupported file format")
	}
}

func TestImportInvalidCSV(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	// Create test categories
	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
		storage.NewCategory(2, "Transport", "uber|taxi|transit"),
	}
	matcher := matcher.New(categories)

	// Test with invalid CSV data
	csvData := `Test Source,invalid-date,Restaurant bill,-1234.56,USD`

	reader := strings.NewReader(csvData)
	info := Import(context.Background(), "test.csv", reader, s, matcher)
	if info.Error == nil {
		t.Errorf("Expected 1 error")
	}
	if !strings.Contains(info.Error.Error(), "parsing time") {
		t.Errorf("Expected parsing error, got: %v", info.Error)
	}
}

func TestImportInvalidJSON(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	// Create test categories
	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
		storage.NewCategory(2, "Transport", "uber|taxi|transit"),
	}
	matcher := matcher.New(categories)

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
	info := Import(context.Background(), "test.json", reader, s, matcher)
	if info.Error == nil {
		t.Errorf("Expected 1 error")
	}
	if !strings.Contains(info.Error.Error(), "parsing time") {
		t.Errorf("Expected parsing error, got: %v", info.Error)
	}
}
