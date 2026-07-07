package expense

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/filter"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestList_FiltersByUser(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "coffee", "USD", -500, now, storage.ChargeType, nil),
		storage.NewExpense(0, "Test Source", "lunch", "USD", -1200, now, storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)

	result, err := svc.List(context.Background(), user.ID(), &filter.ExpenseFilter{}, filter.DefaultSortOptions())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 expenses, got %d", len(result))
	}
}

func TestGroupByYearAndMonth_GroupsCorrectly(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory(context.Background(), user.ID(), "Food", "restaurant|food", 0)
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, storage.ChargeType, &categoryID),
		storage.NewExpense(0, "Test Source", "Uber ride", "USD", -50000, now, storage.ChargeType, nil),
	}

	svc := New(s, logger)

	grouped, years, err := svc.GroupByYearAndMonth(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("GroupByYearAndMonth returned error: %v", err)
	}

	if len(years) != 1 || years[0] != now.Year() {
		t.Fatalf("Expected years=[%d], got %v", now.Year(), years)
	}

	monthExpenses, ok := grouped[now.Year()][now.Month().String()]
	if !ok {
		t.Fatal("Expected expenses for current year/month")
	}

	if len(monthExpenses) != 2 {
		t.Fatalf("Expected 2 expenses, got %d", len(monthExpenses))
	}

	var foundCategorized, foundUncategorized bool
	for _, ev := range monthExpenses {
		if ev.Description() == "Restaurant bill" {
			if ev.CategoryID() != categoryID {
				t.Errorf("Expected category ID %d, got %d", categoryID, ev.CategoryID())
			}
			if ev.Category().Name() != "Food" {
				t.Errorf("Expected category name Food, got %s", ev.Category().Name())
			}
			foundCategorized = true
		}
		if ev.Description() == "Uber ride" {
			foundUncategorized = true
		}
	}

	if !foundCategorized || !foundUncategorized {
		t.Fatalf("Expected both categorized and uncategorized expenses, got categorized=%v uncategorized=%v",
			foundCategorized, foundUncategorized)
	}
}

func TestCreate_InsertsExpense(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	newExpense := storage.NewExpense(0, "Test Source", "New expense", "USD", -1000, time.Now(), storage.ChargeType, nil)

	created, err := svc.Create(context.Background(), user.ID(), newExpense)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if created.Description() != "New expense" {
		t.Fatalf("Expected description 'New expense', got %s", created.Description())
	}

	allExpenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(allExpenses) != 1 {
		t.Fatalf("Expected 1 expense, got %d", len(allExpenses))
	}
}

func TestUpdate_UpdatesFields(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Original Source", "Original description", "EUR", -100000, now, storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	svc := New(s, logger)

	updatedExpense := storage.NewExpense(
		1,
		"Updated Source",
		"Updated description",
		"USD",
		2050,
		now,
		storage.IncomeType,
		nil,
	)

	updated, err := svc.Update(context.Background(), user.ID(), updatedExpense)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if updated != 1 {
		t.Fatalf("Expected 1 row updated, got %d", updated)
	}

	result, err := s.GetExpenseByID(context.Background(), user.ID(), 1)
	if err != nil {
		t.Fatalf("Failed to get expense: %v", err)
	}

	if result.Source() != "Updated Source" {
		t.Errorf("Expected source 'Updated Source', got %s", result.Source())
	}
	if result.Description() != "Updated description" {
		t.Errorf("Expected description 'Updated description', got %s", result.Description())
	}
	if result.Amount() != 2050 {
		t.Errorf("Expected amount 2050, got %d", result.Amount())
	}
}

func TestDelete_RemovesExpense(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Test expense", "USD", -1000, now, storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	svc := New(s, logger)

	err = svc.Delete(context.Background(), user.ID(), 1)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	_, err = s.GetExpenseByID(context.Background(), user.ID(), 1)
	if err == nil {
		t.Fatal("Expected error when getting deleted expense")
	}
}

func TestExport_WritesCSV(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Test expense for export", "USD", -1000, now, storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	svc := New(s, logger)

	var buf bytes.Buffer
	err = svc.Export(context.Background(), user.ID(), &buf)
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "Test expense for export") {
		t.Fatalf("Expected CSV output to contain expense description, got: %s", buf.String())
	}
}
