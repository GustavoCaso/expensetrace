package deletecmd

import (
	"flag"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
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
	if desc != "Delete the expenses DB" {
		t.Errorf("Description() = %v, want %v", desc, "Delete the expenses DB")
	}
}

func TestSetFlags(t *testing.T) {
	cmd := NewCommand()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(fs)

	// Verify that no flags are registered
	if fs.NFlag() != 0 {
		t.Error("SetFlags() registered flags when it shouldn't")
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

	// Test successful deletion
	err := cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Verify tables are deleted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM expenses").Scan(&count)
	if err == nil {
		t.Error("expenses table still exists after deletion")
	}

	err = db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err == nil {
		t.Error("categories table still exists after deletion")
	}
}
