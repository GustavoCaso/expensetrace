package tui

import (
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/storage"
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
	if desc != "Interactive terminal user interface" {
		t.Errorf("Description() = %v, want %v", desc, "Interactive terminal user interface")
	}
}

func TestInitialModel(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	// Create test categories
	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
		storage.NewCategory(2, "Transport", "uber|taxi|transit"),
	}

	for _, c := range categories {
		_, err := s.CreateCategory(c.Name(), c.Pattern())
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	// Create test expenses
	expenses := []storage.Expense{
		storage.NewExpense(int64(1), "test", "EUR", "Restaurant bill", -1000,
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			storage.ChargeType, nil,
		),
		storage.NewExpense(int64(2), "test", "EUR", "Uber drive", -2000,
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			storage.ChargeType, nil,
		),
		storage.NewExpense(int64(2), "test", "EUR", "Salary", 5000,
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			storage.IncomeType, nil,
		),
	}

	_, insertErr := s.InsertExpenses(expenses)
	if insertErr != nil {
		t.Fatalf("Failed to create expenses: %v", insertErr)
	}

	// Test initial model creation
	width := 80
	height := 24
	m, err := initialModel(s, width, height)
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
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	// Create test categories
	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
		storage.NewCategory(2, "Transport", "uber|taxi|transit"),
	}

	for _, c := range categories {
		_, err := s.CreateCategory(c.Name(), c.Pattern())
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	// Create test expenses
	expenses := []storage.Expense{
		storage.NewExpense(int64(1), "test", "EUR", "Restaurant bill", -1000,
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			storage.ChargeType, nil,
		),
		storage.NewExpense(int64(2), "test", "EUR", "Uber drive", -2000,
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			storage.ChargeType, nil,
		),
		storage.NewExpense(int64(2), "test", "EUR", "Salary", 5000,
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			storage.IncomeType, nil,
		),
	}

	_, insertErr := s.InsertExpenses(expenses)
	if insertErr != nil {
		t.Fatalf("Failed to create expenses: %v", insertErr)
	}

	// Test report generation
	reports, err := generateReports(s, time.January, 2024)
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
