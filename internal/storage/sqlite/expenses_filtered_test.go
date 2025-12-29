package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/filter"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func TestGetExpensesFiltered_NoFilters(t *testing.T) {
	stor, user := setupTestStorage(t)
	ctx := context.Background()

	// Insert test expenses
	expenses := []storage.Expense{
		storage.NewExpense(0, "store1", "coffee", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store2", "lunch", "USD", -1200, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "employer", "salary", "USD", 500000, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), storage.IncomeType, nil),
	}

	_, err := stor.InsertExpenses(ctx, user.ID(), expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	// Query with no filters
	emptyFilter := &filter.ExpenseFilter{}
	sortOptions := filter.DefaultSortOptions()

	results, err := stor.GetExpensesFiltered(ctx, user.ID(), emptyFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Verify default sort (date desc) - salary should be last
	if results[0].Description() != "lunch" {
		t.Errorf("expected first result to be 'lunch', got %q", results[0].Description())
	}
}

func TestGetExpensesFiltered_DescriptionFilter(t *testing.T) {
	stor, user := setupTestStorage(t)
	ctx := context.Background()

	// Insert test expenses
	expenses := []storage.Expense{
		storage.NewExpense(0, "store1", "morning coffee", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store2", "afternoon coffee", "USD", -450, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store3", "lunch", "USD", -1200, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := stor.InsertExpenses(ctx, user.ID(), expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	// Filter by description
	desc := "coffee"
	expFilter := &filter.ExpenseFilter{
		Description: &desc,
	}
	sortOptions := filter.DefaultSortOptions()

	results, err := stor.GetExpensesFiltered(ctx, user.ID(), expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Both should contain "coffee"
	for _, r := range results {
		if !strings.Contains(r.Description(), "coffee") {
			t.Errorf("expected description to contain 'coffee', got %q", r.Description())
		}
	}
}

func TestGetExpensesFiltered_SourceFilter(t *testing.T) {
	stor, user := setupTestStorage(t)
	ctx := context.Background()

	expenses := []storage.Expense{
		storage.NewExpense(0, "visa", "coffee", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "mastercard", "lunch", "USD", -1200, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "visa", "dinner", "USD", -2000, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := stor.InsertExpenses(ctx, user.ID(), expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	source := "visa"
	expFilter := &filter.ExpenseFilter{
		Source: &source,
	}
	sortOptions := filter.DefaultSortOptions()

	results, err := stor.GetExpensesFiltered(ctx, user.ID(), expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Both should have source "visa"
	for _, r := range results {
		if r.Source() != "visa" {
			t.Errorf("expected source to be 'visa', got %q", r.Source())
		}
	}
}

func TestGetExpensesFiltered_AmountRangeFilter(t *testing.T) {
	stor, user := setupTestStorage(t)
	ctx := context.Background()

	expenses := []storage.Expense{
		storage.NewExpense(0, "store", "small", "USD", -300, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "medium", "USD", -800, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "large", "USD", -1500, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := stor.InsertExpenses(ctx, user.ID(), expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	min := int64(-1000)
	max := int64(-500)
	expFilter := &filter.ExpenseFilter{
		AmountMin: &min,
		AmountMax: &max,
	}
	sortOptions := filter.DefaultSortOptions()

	results, err := stor.GetExpensesFiltered(ctx, user.ID(), expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Description() != "medium" {
		t.Errorf("expected 'medium', got %q", results[0].Description())
	}
}

func TestGetExpensesFiltered_DateRangeFilter(t *testing.T) {
	stor, user := setupTestStorage(t)
	ctx := context.Background()

	expenses := []storage.Expense{
		storage.NewExpense(0, "store", "jan", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "feb", "USD", -600, time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "mar", "USD", -700, time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := stor.InsertExpenses(ctx, user.ID(), expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	from := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)
	expFilter := &filter.ExpenseFilter{
		DateFrom: &from,
		DateTo:   &to,
	}
	sortOptions := filter.DefaultSortOptions()

	results, err := stor.GetExpensesFiltered(ctx, user.ID(), expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Description() != "feb" {
		t.Errorf("expected 'feb', got %q", results[0].Description())
	}
}

func TestGetExpensesFiltered_CombinedFilters(t *testing.T) {
	stor, user := setupTestStorage(t)
	ctx := context.Background()

	expenses := []storage.Expense{
		storage.NewExpense(0, "visa", "coffee shop", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "visa", "coffee shop", "USD", -600, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "mastercard", "coffee shop", "USD", -550, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "visa", "restaurant", "USD", -1200, time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := stor.InsertExpenses(ctx, user.ID(), expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	// Filter: visa + coffee + amount < -520
	source := "visa"
	desc := "coffee"
	max := int64(-520)
	expFilter := &filter.ExpenseFilter{
		Source:      &source,
		Description: &desc,
		AmountMax:   &max,
	}
	sortOptions := filter.DefaultSortOptions()

	results, err := stor.GetExpensesFiltered(ctx, user.ID(), expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if len(results) > 0 {
		if results[0].Amount() != -600 {
			t.Errorf("expected amount -600, got %d", results[0].Amount())
		}
	}
}

func TestGetExpensesFiltered_SortByDateAsc(t *testing.T) {
	stor, user := setupTestStorage(t)
	ctx := context.Background()

	expenses := []storage.Expense{
		storage.NewExpense(0, "store", "third", "USD", -500, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "first", "USD", -600, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "second", "USD", -700, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := stor.InsertExpenses(ctx, user.ID(), expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	expFilter := &filter.ExpenseFilter{}
	sortOptions := &filter.SortOptions{
		Field:     filter.SortByDate,
		Direction: filter.SortAsc,
	}

	results, err := stor.GetExpensesFiltered(ctx, user.ID(), expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Description() != "first" {
		t.Errorf("expected 'first' at position 0, got %q", results[0].Description())
	}
	if results[1].Description() != "second" {
		t.Errorf("expected 'second' at position 1, got %q", results[1].Description())
	}
	if results[2].Description() != "third" {
		t.Errorf("expected 'third' at position 2, got %q", results[2].Description())
	}
}

func TestGetExpensesFiltered_SortByAmountDesc(t *testing.T) {
	stor, user := setupTestStorage(t)
	ctx := context.Background()

	expenses := []storage.Expense{
		storage.NewExpense(0, "store", "medium", "USD", -800, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "small", "USD", -500, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "large", "USD", -1200, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := stor.InsertExpenses(ctx, user.ID(), expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	expFilter := &filter.ExpenseFilter{}
	sortOptions := &filter.SortOptions{
		Field:     filter.SortByAmount,
		Direction: filter.SortDesc,
	}

	results, err := stor.GetExpensesFiltered(ctx, user.ID(), expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Desc order: largest (least negative) first
	if results[0].Description() != "small" {
		t.Errorf("expected 'small' at position 0, got %q", results[0].Description())
	}
	if results[1].Description() != "medium" {
		t.Errorf("expected 'medium' at position 1, got %q", results[1].Description())
	}
	if results[2].Description() != "large" {
		t.Errorf("expected 'large' at position 2, got %q", results[2].Description())
	}
}

func TestGetExpensesFiltered_UserIsolation(t *testing.T) {
	stor, user1 := setupTestStorage(t)
	ctx := context.Background()

	// Create second user
	user2, err := stor.CreateUser(ctx, "testuser2", "password2")
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	// Insert expenses for user 1
	user1Expenses := []storage.Expense{
		storage.NewExpense(0, "store", "user1 expense1", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "user1 expense2", "USD", -600, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}
	_, err = stor.InsertExpenses(ctx, user1.ID(), user1Expenses)
	if err != nil {
		t.Fatalf("failed to insert user1 expenses: %v", err)
	}

	// Insert expenses for user 2
	user2Expenses := []storage.Expense{
		storage.NewExpense(0, "store", "user2 expense1", "USD", -700, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}
	_, err = stor.InsertExpenses(ctx, user2.ID(), user2Expenses)
	if err != nil {
		t.Fatalf("failed to insert user2 expenses: %v", err)
	}

	// Query for user 1 - should only see user 1's expenses
	expFilter := &filter.ExpenseFilter{}
	sortOptions := filter.DefaultSortOptions()

	results, err := stor.GetExpensesFiltered(ctx, user1.ID(), expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("user1: expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if !strings.Contains(r.Description(), "user1") {
			t.Errorf("user1 query returned user2's expense: %q", r.Description())
		}
	}

	// Query for user 2 - should only see user 2's expenses
	results, err = stor.GetExpensesFiltered(ctx, user2.ID(), expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("user2: expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && !strings.Contains(results[0].Description(), "user2") {
		t.Errorf("user2 query returned user1's expense: %q", results[0].Description())
	}
}
