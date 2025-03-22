package tui

import (
	"database/sql"
	"flag"
	"testing"
	"time"

	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := expenseDB.CreateExpenseTable(db); err != nil {
		t.Fatalf("Failed to create expense table: %v", err)
	}

	if err := expenseDB.CreateCategoriesTable(db); err != nil {
		t.Fatalf("Failed to create categories table: %v", err)
	}

	return db
}

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()
	if cmd == nil {
		t.Error("NewCommand() returned nil")
	}
}

func TestDescription(t *testing.T) {
	cmd := NewCommand()
	desc := cmd.Description()
	if desc != "Interactive terminal user interface" {
		t.Errorf("Description() = %v, want %v", desc, "Interactive terminal user interface")
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

func TestInitialModel(t *testing.T) {
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

	// Test initial model creation
	width := 80
	height := 24
	m, err := initialModel(db, width, height)
	if err != nil {
		t.Fatalf("initialModel() error = %v", err)
	}

	// Verify model fields
	if m.width != width {
		t.Errorf("width = %v, want %v", m.width, width)
	}
	if m.height != height {
		t.Errorf("height = %v, want %v", m.height, height)
	}
	if m.focusMode != focusedMain {
		t.Errorf("focusMode = %v, want %v", m.focusMode, focusedMain)
	}
	if len(m.reports) == 0 {
		t.Error("reports is empty")
	}
}

func TestGenerateReports(t *testing.T) {
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

	// Test report generation
	reports, err := generateReports(db, time.January, 2024)
	if err != nil {
		t.Fatalf("generateReports() error = %v", err)
	}

	if len(reports) == 0 {
		t.Error("reports is empty")
	}

	// Verify report content
	report := reports[0].report
	if report.Title != "January 2024" {
		t.Errorf("report.Title = %v, want %v", report.Title, "January 2024")
	}
	if report.Spending != -3000 {
		t.Errorf("report.Spending = %v, want %v", report.Spending, -3000)
	}
	if report.Income != 5000 {
		t.Errorf("report.Income = %v, want %v", report.Income, 5000)
	}
	if report.Savings != 2000 {
		t.Errorf("report.Savings = %v, want %v", report.Savings, 2000)
	}
}

func TestFocusModeToggle(t *testing.T) {
	m := model{
		focusMode: focusedMain,
	}

	// Test toggle from main to detail
	m.focusMode = m.focusModeToggle()
	if m.focusMode != focusedDetail {
		t.Errorf("focusMode = %v, want %v", m.focusMode, focusedDetail)
	}

	// Test toggle from detail to main
	m.focusMode = m.focusModeToggle()
	if m.focusMode != focusedMain {
		t.Errorf("focusMode = %v, want %v", m.focusMode, focusedMain)
	}
}

func TestSetDimensions(t *testing.T) {
	m := model{
		width:  80,
		height: 24,
	}

	// Test setting width
	m.SetWidth(100)
	if m.width != 100 {
		t.Errorf("width = %v, want %v", m.width, 100)
	}

	// Test setting height
	m.SetHeight(30)
	if m.height != 30 {
		t.Errorf("height = %v, want %v", m.height, 30)
	}
}
