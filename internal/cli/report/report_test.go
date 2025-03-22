package report

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()
	if cmd == nil {
		t.Error("NewCommand() returned nil")
	}
}

func TestDescription(t *testing.T) {
	cmd := NewCommand()
	desc := cmd.Description()
	if desc != "Displays the expenses information for selected date ranges" {
		t.Errorf("Description() = %v, want %v", desc, "Displays the expenses information for selected date ranges")
	}
}

func TestSetFlags(t *testing.T) {
	cmd := NewCommand()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(fs)

	// Test month flag
	if fs.Lookup("month") == nil {
		t.Error("Month flag not registered")
	}

	// Test year flag
	if fs.Lookup("year") == nil {
		t.Error("Year flag not registered")
	}

	// Test verbose flag
	if fs.Lookup("v") == nil {
		t.Error("Verbose flag not registered")
	}
}

func TestRun(t *testing.T) {
	db := testutil.SetupTestDB(t)

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

	// Create test expenses
	expenses := []*expenseDB.Expense{
		{
			Description: "Restaurant bill",
			Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Amount:      -1000,
			Type:        expenseDB.ChargeType,
		},
		{
			Description: "Uber ride",
			Date:        time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			Amount:      -2000,
			Type:        expenseDB.ChargeType,
		},
		{
			Description: "Salary",
			Date:        time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			Amount:      5000,
			Type:        expenseDB.IncomeType,
		},
	}

	errs := expenseDB.InsertExpenses(db, expenses)
	if len(errs) > 0 {
		t.Fatalf("Failed to create expenses: %v", errs)
	}

	// Create category matcher
	matcher := category.NewMatcher(categories)

	// Create command
	cmd := NewCommand()

	// Test default report (previous month)
	month = -1
	year = -1
	err := cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Test monthly report
	month = 1
	year = 2024
	err = cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Test yearly report
	month = 0
	year = 2024
	err = cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Test verbose output
	month = 1
	year = 2024
	verbose = true
	err = cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Test invalid month
	month = 13
	year = 2024
	err = cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Test invalid year
	month = 1
	year = 0
	err = cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
}

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		value       interface{}
		expectError bool
	}{
		{
			name:        "Valid template",
			template:    "report.tmpl",
			value:       report.Report{},
			expectError: false,
		},
		{
			name:        "Invalid template",
			template:    "nonexistent.tmpl",
			value:       struct{}{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := renderTemplate(os.Stdout, tt.template, tt.value)
			if tt.expectError && err == nil {
				t.Error("renderTemplate() expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("renderTemplate() error = %v", err)
			}
		})
	}
}
