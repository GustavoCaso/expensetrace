package category

import (
	"bytes"
	"database/sql"
	"flag"
	"os"
	"strings"
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

	// Check if action flag is registered
	actionFlag := fs.Lookup("a")
	if actionFlag == nil {
		t.Fatal("Expected action flag to be registered")
	}

	if actionFlag.DefValue != "inspect" {
		t.Errorf("Action default value = %q, want inspect", actionFlag.DefValue)
	}

	// Check if output location flag is registered
	outputFlag := fs.Lookup("o")
	if outputFlag == nil {
		t.Fatal("Expected output location flag to be registered")
	}

	if outputFlag.DefValue != "" {
		t.Errorf("Output location default value = %q, want empty string", outputFlag.DefValue)
	}
}

func TestInspect(t *testing.T) {
	testCases := []struct {
		name     string
		expenses []*expenseDB.Expense
		want     string
	}{
		{
			name:     "No expenses",
			expenses: []*expenseDB.Expense{},
			want:     "",
		},
		{
			name: "Single expense",
			expenses: []*expenseDB.Expense{
				{
					Description: "Restaurant",
					Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					Amount:      -1000,
					Type:        expenseDB.ChargeType,
				},
			},
			want: "Restaurant -> 1\n\t[2024-01-01] -10,00€\n\nThere are a total of 1 uncategorized expenses",
		},
		{
			name: "Multiple expenses with same description",
			expenses: []*expenseDB.Expense{
				{
					Description: "Restaurant",
					Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					Amount:      -1000,
					Type:        expenseDB.ChargeType,
				},
				{
					Description: "Restaurant",
					Date:        time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
					Amount:      -2000,
					Type:        expenseDB.ChargeType,
				},
			},
			want: "Restaurant -> 2\n\t[2024-01-01] -10,00€\n\t[2024-01-02] -20,00€\n\nThere are a total of 2 uncategorized expenses",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			inspect(&buf, tc.expenses)
			got := strings.TrimSpace(buf.String())
			want := strings.TrimSpace(tc.want)
			if got != want {
				t.Errorf("inspect() output = %q, want %q", got, want)
			}
		})
	}
}

func TestRecategorize(t *testing.T) {
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
			Description: "restaurant",
			Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Amount:      -1000,
			Type:        expenseDB.ChargeType,
		},
		{
			Description: "uber",
			Date:        time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			Amount:      -2000,
			Type:        expenseDB.ChargeType,
		},
		{
			Description: "Other expense",
			Date:        time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			Amount:      -3000,
			Type:        expenseDB.ChargeType,
		},
	}

	errs := expenseDB.InsertExpenses(db, expenses)
	if len(errs) > 0 {
		t.Fatalf("Failed to create expenses: %v", errs)
	}

	// Create category matcher
	matcher := category.NewMatcher(categories)

	// Run recategorize
	recategorize(db, matcher, expenses)

	// Check if expenses were categorized correctly
	updatedExpenses, err := expenseDB.GetExpenses(db)
	if err != nil {
		t.Fatalf("Failed to get updated expenses: %v", err)
	}

	expectedCategories := map[string]sql.NullInt64{
		"restaurant":    {Int64: int64(1), Valid: true}, // Food category
		"uber":          {Int64: int64(2), Valid: true}, // Transport category
		"Other expense": {Int64: 0, Valid: false},       // No category
	}

	for _, e := range updatedExpenses {
		expectedCategoryID := expectedCategories[e.Description]

		if e.CategoryID.Int64 != expectedCategoryID.Int64 {
			t.Errorf(
				"Expense %q has category ID %d, want %d",
				e.Description,
				e.CategoryID.Int64,
				expectedCategoryID.Int64,
			)
		}
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
			Description: "restaurant",
			Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Amount:      -1000,
			Type:        expenseDB.ChargeType,
		},
		{
			Description: "uber",
			Date:        time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			Amount:      -2000,
			Type:        expenseDB.ChargeType,
		},
		{
			Description: "Other expense",
			Date:        time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			Amount:      -3000,
			Type:        expenseDB.ChargeType,
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

	// Test inspect action
	actionFlag = "inspect"
	outputLocation = "test_output.txt"
	err := cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	err = os.Remove(outputLocation)
	if err != nil {
		t.Errorf("failed to delete test_output.txt= %v", err)
	}

	// Test recategorize action
	actionFlag = "recategorize"
	err = cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Test migrate action
	actionFlag = "migrate"
	err = cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Test invalid action
	actionFlag = "invalid"
	err = cmd.Run(db, matcher)
	if err == nil {
		t.Error("Run() expected error for invalid action, got nil")
	}

	// Test invalid output location
	actionFlag = "inspect"
	outputLocation = "/invalid/path/test_output.txt"
	err = cmd.Run(db, matcher)
	if err == nil {
		t.Error("Run() expected error for invalid output location, got nil")
	}
}
