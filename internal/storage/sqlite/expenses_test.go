package sqlite

import (
	"errors"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func TestExpenseStorage(t *testing.T) {
	stor := setupTestStorage(t)

	// Test creating expenses
	now := time.Now()
	testExpenses := []storage.Expense{
		storage.NewExpense(0, "Test Bank", "Coffee shop", "USD", -500, now, storage.ChargeType, nil),
		storage.NewExpense(0, "Test Bank", "Salary deposit", "USD", 500000, now, storage.IncomeType, nil),
	}

	insertedCount, err := stor.InsertExpenses(testExpenses)
	if err != nil {
		t.Fatalf("Failed to insert expenses: %v", err)
	}
	if insertedCount != 2 {
		t.Errorf("Expected 2 expenses inserted, got %d", insertedCount)
	}

	// Test getting all expenses
	allExpenses, err := stor.GetExpenses()
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}
	if len(allExpenses) != 2 {
		t.Errorf("Expected 2 expenses, got %d", len(allExpenses))
	}

	// Test getting expense by ID
	expense, err := stor.GetExpenseByID(1)
	if err != nil {
		t.Fatalf("Failed to get expense by ID: %v", err)
	}
	if expense.Description() != "Coffee shop" {
		t.Errorf("Expected description 'Coffee shop', got '%s'", expense.Description())
	}

	// Test getting first expense
	firstExpense, err := stor.GetFirstExpense()
	if err != nil {
		t.Fatalf("Failed to get first expense: %v", err)
	}
	if firstExpense.ID() != 1 {
		t.Errorf("Expected first expense ID 1, got %d", firstExpense.ID())
	}

	// Test date range query
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)
	rangeExpenses, err := stor.GetExpensesFromDateRange(yesterday, tomorrow)
	if err != nil {
		t.Fatalf("Failed to get expenses from date range: %v", err)
	}
	if len(rangeExpenses) != 2 {
		t.Errorf("Expected 2 expenses in date range, got %d", len(rangeExpenses))
	}

	// Test search
	searchResults, err := stor.SearchExpenses("coffee")
	if err != nil {
		t.Fatalf("Failed to search expenses: %v", err)
	}
	if len(searchResults) != 1 {
		t.Errorf("Expected 1 search result, got %d", len(searchResults))
	}

	// Test update expense
	updatedExpense := storage.NewExpense(1, "Updated Bank", "Updated coffee", "EUR", -600, now, storage.ChargeType, nil)
	updateCount, err := stor.UpdateExpense(updatedExpense)
	if err != nil {
		t.Fatalf("Failed to update expense: %v", err)
	}
	if updateCount != 1 {
		t.Errorf("Expected 1 expense updated, got %d", updateCount)
	}

	// Verify update
	updated, err := stor.GetExpenseByID(1)
	if err != nil {
		t.Fatalf("Failed to get updated expense: %v", err)
	}
	if updated.Description() != "Updated coffee" {
		t.Errorf("Expected updated description 'Updated coffee', got '%s'", updated.Description())
	}
	if updated.Currency() != "EUR" {
		t.Errorf("Expected updated currency 'EUR', got '%s'", updated.Currency())
	}

	// Test delete expense
	deleteCount, err := stor.DeleteExpense(2)
	if err != nil {
		t.Fatalf("Failed to delete expense: %v", err)
	}
	if deleteCount != 1 {
		t.Errorf("Expected 1 expense deleted, got %d", deleteCount)
	}

	// Verify deletion
	remainingExpenses, err := stor.GetExpenses()
	if err != nil {
		t.Fatalf("Failed to get remaining expenses: %v", err)
	}
	if len(remainingExpenses) != 1 {
		t.Errorf("Expected 1 remaining expense, got %d", len(remainingExpenses))
	}

	// Verify we get an error when fething a non existing expense
	_, err = stor.GetExpenseByID(2)
	if err == nil {
		t.Fatal("Expected error when fetching a non existng expense")
	}

	if !errors.Is(err, &storage.NotFoundError{}) {
		t.Fatal("Expected error to be of type storage.NotFoundError")
	}
}

func TestExpenseWithCategories(t *testing.T) {
	stor := setupTestStorage(t)

	// Create a category first
	categoryID, err := stor.CreateCategory("Food", "restaurant|coffee")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	// Create expense with category
	now := time.Now()
	expenseWithCategory := storage.NewExpense(
		0,
		"Test Bank",
		"Restaurant dinner",
		"USD",
		-2500,
		now,
		storage.ChargeType,
		&categoryID,
	)
	expenseWithoutCategory := storage.NewExpense(
		0,
		"Test Bank",
		"Random purchase",
		"USD",
		-1000,
		now,
		storage.ChargeType,
		nil,
	)

	expenses := []storage.Expense{expenseWithCategory, expenseWithoutCategory}
	_, err = stor.InsertExpenses(expenses)
	if err != nil {
		t.Fatalf("Failed to insert expenses: %v", err)
	}

	// Test getting expenses by category
	categoryExpenses, err := stor.GetExpensesByCategory(categoryID)
	if err != nil {
		t.Fatalf("Failed to get expenses by category: %v", err)
	}
	if len(categoryExpenses) != 1 {
		t.Errorf("Expected 1 expense in category, got %d", len(categoryExpenses))
	}
	if categoryExpenses[0].Description() != "Restaurant dinner" {
		t.Errorf("Expected description 'Restaurant dinner', got '%s'", categoryExpenses[0].Description())
	}

	// Test getting expenses without category
	uncategorizedExpenses, err := stor.GetExpensesWithoutCategory()
	if err != nil {
		t.Fatalf("Failed to get uncategorized expenses: %v", err)
	}
	if len(uncategorizedExpenses) != 1 {
		t.Errorf("Expected 1 uncategorized expense, got %d", len(uncategorizedExpenses))
	}
	if uncategorizedExpenses[0].Description() != "Random purchase" {
		t.Errorf("Expected description 'Random purchase', got '%s'", uncategorizedExpenses[0].Description())
	}
}
