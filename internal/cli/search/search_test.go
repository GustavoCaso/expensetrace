package search

import (
	"bytes"
	"flag"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
	"github.com/fatih/color"
	_ "github.com/mattn/go-sqlite3"

	categoryPkg "github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
)

func TestSetFlags(t *testing.T) {
	cmd := NewCommand()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(fs)

	// Check if keyword flag is registered
	keywordFlag := fs.Lookup("k")
	if keywordFlag == nil {
		t.Fatal("Expected keyword flag to be registered")
	}

	if keywordFlag.DefValue != "" {
		t.Errorf("Keyword default value = %q, want empty string", keywordFlag.DefValue)
	}

	// Check if verbose flag is registered
	verboseFlag := fs.Lookup("v")
	if verboseFlag == nil {
		t.Fatal("Expected verbose flag to be registered")
	}

	if verboseFlag.DefValue != "false" {
		t.Errorf("Verbose default value = %q, want false", verboseFlag.DefValue)
	}
}

func TestCategoryDisplay(t *testing.T) {
	// Disable color for testing
	color.NoColor = true

	testCases := []struct {
		name     string
		category category
		verbose  bool
		want     string
	}{
		{
			name: "Charge with verbose=false",
			category: category{
				name:         "Food",
				amount:       -1000,
				categoryType: db.ChargeType,
				expenses: []*db.Expense{
					{
						Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						Description: "Restaurant",
						Amount:      -1000,
					},
				},
			},
			verbose: false,
			want:    "Food: -10,00€",
		},
		{
			name: "Income with verbose=false",
			category: category{
				name:         "Salary",
				amount:       100000,
				categoryType: db.IncomeType,
				expenses: []*db.Expense{
					{
						Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						Description: "Monthly salary",
						Amount:      100000,
					},
				},
			},
			verbose: false,
			want:    "Salary: 1.000,00€",
		},
		{
			name: "Charge with verbose=true",
			category: category{
				name:         "Food",
				amount:       -1000,
				categoryType: db.ChargeType,
				expenses: []*db.Expense{
					{
						Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						Description: "Restaurant",
						Amount:      -1000,
					},
				},
			},
			verbose: true,
			want:    "Food: -10,00€\n2024-01-01 Restaurant -10,00€\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.category.Display(tc.verbose)
			if got != tc.want {
				t.Errorf("Display(%v) = %q, want %q", tc.verbose, got, tc.want)
			}
		})
	}
}

func TestColorOutput(t *testing.T) {
	// Enable color for this test
	color.NoColor = false

	testCases := []struct {
		name     string
		text     string
		options  []string
		wantFunc func(string) string
	}{
		{
			name:    "Red text",
			text:    "test",
			options: []string{"red"},
			wantFunc: func(text string) string {
				return color.New(color.FgHiRed).Sprint(text)
			},
		},
		{
			name:    "Green bold text",
			text:    "test",
			options: []string{"green", "bold"},
			wantFunc: func(text string) string {
				return color.New(color.FgGreen, color.Bold).Sprint(text)
			},
		},
		{
			name:    "Invalid color option",
			text:    "test",
			options: []string{"invalid"},
			wantFunc: func(text string) string {
				return color.New().Sprint(text)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := colorOutput(tc.text, tc.options...)
			want := tc.wantFunc(tc.text)
			if got != want {
				t.Errorf("colorOutput(%q, %v) = %q, want %q", tc.text, tc.options, got, want)
			}
		})
	}
}

func TestRenderTemplate(t *testing.T) {
	// Test basic template rendering
	var buf bytes.Buffer
	data := report{
		Categories: map[string]category{
			"Food": {
				name:         "Food",
				amount:       -1000,
				categoryType: db.ChargeType,
				expenses: []*db.Expense{
					{
						Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						Description: "Restaurant",
						Amount:      -1000,
					},
				},
			},
		},
		Verbose: false,
	}

	err := renderTemplate(&buf, "report.tmpl", data)
	if err != nil {
		t.Fatalf("renderTemplate() error = %v", err)
	}

	// Note: We can't test the exact output since it depends on the template file,
	// but we can at least verify that some output was generated
	if buf.Len() == 0 {
		t.Error("renderTemplate() produced no output")
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
	matcher := categoryPkg.NewMatcher(categories)

	// Create command
	cmd := NewCommand()

	// Test without keyword
	keyword = ""
	err := cmd.Run(db, matcher)
	if err == nil {
		t.Error("Run() expected error for empty keyword, got nil")
	}

	// Test with keyword
	keyword = "restaurant"
	err = cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Test with verbose output
	keyword = "uber"
	verbose = true
	err = cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Test with non-existent keyword
	keyword = "nonexistent"
	err = cmd.Run(db, matcher)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
}

func TestExpenseCategory(t *testing.T) {
	tests := []struct {
		name     string
		expense  *expenseDB.Expense
		expected string
	}{
		{
			name: "Uncategorized income",
			expense: &expenseDB.Expense{
				CategoryID: nil,
				Type:       expenseDB.IncomeType,
			},
			expected: "uncategorized income",
		},
		{
			name: "Uncategorized charge",
			expense: &expenseDB.Expense{
				CategoryID: nil,
				Type:       expenseDB.ChargeType,
			},
			expected: "uncategorized charge",
		},
		{
			name: "Categorized expense",
			expense: &expenseDB.Expense{
				CategoryID: intPtr(1),
				Type:       expenseDB.ChargeType,
			},
			expected: "Needs to fix this",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expeseCategory(tt.expense)
			if result != tt.expected {
				t.Errorf("expeseCategory() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func intPtr(x int) *int {
	return &x
}
