package db

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/GustavoCaso/expensetrace/internal/logger"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	testLogger := logger.New(logger.Config{
		Level:  logger.LevelInfo,
		Format: logger.FormatText,
		Output: "discard",
	})
	err = ApplyMigrations(database, testLogger)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	_, err = database.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable PRAGMA: %v", err)
	}

	t.Cleanup(func() {
		if err = database.Close(); err != nil {
			t.Errorf("Failed to close test expenses database: %v", err)
		}
	})

	return database
}

func TestCreateExpenseTable(t *testing.T) {
	database := setupTestDB(t)

	_, err := database.Exec(
		"INSERT INTO expenses(source, amount, description, expense_type, date, currency, category_id) VALUES(?, ?, ?, ?, ?, ?, ?)",
		"test",
		1000,
		"Test expense",
		ChargeType,
		time.Now().Unix(),
		"USD",
		nil,
	)
	if err != nil {
		t.Errorf("Failed to insert test expense: %v", err)
	}
}

func TestInsertExpenses(t *testing.T) {
	database := setupTestDB(t)

	now := time.Now()
	expenses := []*Expense{
		{
			Source:      "test1",
			Amount:      1000,
			Description: "Test expense 1",
			Type:        ChargeType,
			Date:        now,
			Currency:    "USD",
		},
		{
			Source:      "test2",
			Amount:      2000,
			Description: "Test expense 2",
			Type:        IncomeType,
			Date:        now,
			Currency:    "USD",
		},
	}

	_, err := InsertExpenses(database, expenses)
	if err != nil {
		t.Errorf("Failed to insert expenses: %v", err)
	}

	got, err := GetExpenses(database)
	if err != nil {
		t.Errorf("Failed to get expenses: %v", err)
	}

	if len(got) != len(expenses) {
		t.Errorf("Expected %d expenses, got %d", len(expenses), len(got))
	}

	for i, exp := range got {
		if exp.Source != expenses[i].Source {
			t.Errorf("Expense[%d].Source = %v, want %v", i, exp.Source, expenses[i].Source)
		}
		if exp.Amount != expenses[i].Amount {
			t.Errorf("Expense[%d].Amount = %v, want %v", i, exp.Amount, expenses[i].Amount)
		}
		if exp.Description != expenses[i].Description {
			t.Errorf("Expense[%d].Description = %v, want %v", i, exp.Description, expenses[i].Description)
		}
		if exp.Type != expenses[i].Type {
			t.Errorf("Expense[%d].Type = %v, want %v", i, exp.Type, expenses[i].Type)
		}
		if exp.Currency != expenses[i].Currency {
			t.Errorf("Expense[%d].Currency = %v, want %v", i, exp.Currency, expenses[i].Currency)
		}
		if exp.CategoryID != expenses[i].CategoryID {
			t.Errorf("Expense[%d].CategoryID = %v, want %v", i, exp.CategoryID, expenses[i].CategoryID)
		}
	}
}

func TestGetExpenses(t *testing.T) {
	database := setupTestDB(t)

	now := time.Now()
	testExpenses := []struct {
		source      string
		amount      int64
		description string
		expenseType ExpenseType
		currency    string
		categoryID  sql.NullInt64
	}{
		{"test1", 1000, "Test expense 1", ChargeType, "USD", sql.NullInt64{Int64: 0, Valid: false}},
		{"test2", 2000, "Test expense 2", IncomeType, "USD", sql.NullInt64{Int64: 0, Valid: false}},
		{"test3", 3000, "Test expense 3", ChargeType, "EUR", sql.NullInt64{Int64: 1, Valid: true}},
	}

	_, err := CreateCategory(database, "Test", "*")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	for _, exp := range testExpenses {
		_, err = database.Exec(
			"INSERT INTO expenses(source, amount, description, expense_type, date, currency, category_id) VALUES(?, ?, ?, ?, ?, ?, ?)",
			exp.source,
			exp.amount,
			exp.description,
			exp.expenseType,
			now.Unix(),
			exp.currency,
			exp.categoryID,
		)
		if err != nil {
			t.Fatalf("Failed to insert test expense: %v", err)
		}
	}

	expenses, err := GetExpenses(database)
	if err != nil {
		t.Errorf("Failed to get expenses: %v", err)
	}

	if len(expenses) != len(testExpenses) {
		t.Errorf("Expected %d expenses, got %d", len(testExpenses), len(expenses))
	}

	for i, exp := range expenses {
		if exp.Source != testExpenses[i].source {
			t.Errorf("Expense[%d].Source = %v, want %v", i, exp.Source, testExpenses[i].source)
		}
		if exp.Amount != testExpenses[i].amount {
			t.Errorf("Expense[%d].Amount = %v, want %v", i, exp.Amount, testExpenses[i].amount)
		}
		if exp.Description != testExpenses[i].description {
			t.Errorf("Expense[%d].Description = %v, want %v", i, exp.Description, testExpenses[i].description)
		}
		if exp.Type != testExpenses[i].expenseType {
			t.Errorf("Expense[%d].Type = %v, want %v", i, exp.Type, testExpenses[i].expenseType)
		}
		if exp.Currency != testExpenses[i].currency {
			t.Errorf("Expense[%d].Currency = %v, want %v", i, exp.Currency, testExpenses[i].currency)
		}

		if exp.CategoryID.Int64 != testExpenses[i].categoryID.Int64 {
			t.Errorf("Expense[%d].CategoryID = %v, want %v", i, exp.CategoryID.Int64, testExpenses[i].categoryID.Int64)
		}
	}
}

func TestGetExpensesFromDateRange(t *testing.T) {
	database := setupTestDB(t)

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	testExpenses := []struct {
		source      string
		amount      int64
		description string
		expenseType ExpenseType
		date        time.Time
		currency    string
		categoryID  sql.NullInt64
	}{
		{"test1", 1000, "Test expense 1", ChargeType, yesterday, "USD", sql.NullInt64{Int64: 0, Valid: false}},
		{"test2", 2000, "Test expense 2", IncomeType, now, "USD", sql.NullInt64{Int64: 0, Valid: false}},
		{"test3", 3000, "Test expense 3", ChargeType, tomorrow, "EUR", sql.NullInt64{Int64: 1, Valid: true}},
	}

	_, err := CreateCategory(database, "Test", "*")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	for _, exp := range testExpenses {
		_, err = database.Exec(
			"INSERT INTO expenses(source, amount, description, expense_type, date, currency, category_id) VALUES(?, ?, ?, ?, ?, ?, ?)",
			exp.source,
			exp.amount,
			exp.description,
			exp.expenseType,
			exp.date.Unix(),
			exp.currency,
			exp.categoryID,
		)
		if err != nil {
			t.Fatalf("Failed to insert test expense: %v", err)
		}
	}

	expenses, err := GetExpensesFromDateRange(database, yesterday, tomorrow)
	if err != nil {
		t.Errorf("Failed to get expenses from date range: %v", err)
	}

	if len(expenses) != 3 {
		t.Errorf("Expected 3 expenses, got %d", len(expenses))
	}
}

func TestGetExpensesWithoutCategory(t *testing.T) {
	database := setupTestDB(t)

	now := time.Now()
	testExpenses := []struct {
		source      string
		amount      int64
		description string
		expenseType ExpenseType
		currency    string
		categoryID  sql.NullInt64
	}{
		{"test1", 1000, "Test expense 1", ChargeType, "USD", sql.NullInt64{Int64: 0, Valid: false}},
		{"test2", 2000, "Test expense 2", IncomeType, "USD", sql.NullInt64{Int64: 1, Valid: true}},
		{"test3", 3000, "Test expense 3", ChargeType, "EUR", sql.NullInt64{Int64: 0, Valid: false}},
	}

	_, err := CreateCategory(database, "Test", "*")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	for _, exp := range testExpenses {
		_, err = database.Exec(
			"INSERT INTO expenses(source, amount, description, expense_type, date, currency, category_id) VALUES(?, ?, ?, ?, ?, ?, ?)",
			exp.source,
			exp.amount,
			exp.description,
			exp.expenseType,
			now.Unix(),
			exp.currency,
			exp.categoryID,
		)
		if err != nil {
			t.Fatalf("Failed to insert test expense: %v", err)
		}
	}

	expenses, err := GetExpensesWithoutCategory(database)
	if err != nil {
		t.Errorf("Failed to get expenses without category: %v", err)
	}

	if len(expenses) != 2 {
		t.Errorf("Expected 2 expenses without category, got %d", len(expenses))
	}

	for _, exp := range expenses {
		if exp.CategoryID.Valid {
			t.Errorf("Expected CategoryID to be NULL, got %+v", exp.CategoryID)
		}
	}
}

func TestSearchExpenses(t *testing.T) {
	database := setupTestDB(t)

	now := time.Now()
	testExpenses := []struct {
		source      string
		amount      int64
		description string
		expenseType ExpenseType
		currency    string
		categoryID  sql.NullInt64
	}{
		{"test1", 1000, "Test expense 1", ChargeType, "USD", sql.NullInt64{Int64: 0, Valid: false}},
		{"test2", 2000, "Test expense 2", IncomeType, "USD", sql.NullInt64{Int64: 1, Valid: true}},
		{"test3", 3000, "Test expense 3", ChargeType, "EUR", sql.NullInt64{Int64: 0, Valid: false}},
	}

	_, err := CreateCategory(database, "Test", "*")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	for _, exp := range testExpenses {
		_, err = database.Exec(
			"INSERT INTO expenses(source, amount, description, expense_type, date, currency, category_id) VALUES(?, ?, ?, ?, ?, ?, ?)",
			exp.source,
			exp.amount,
			exp.description,
			exp.expenseType,
			now.Unix(),
			exp.currency,
			exp.categoryID,
		)
		if err != nil {
			t.Fatalf("Failed to insert test expense: %v", err)
		}
	}

	expenses, err := SearchExpenses(database, "Test")
	if err != nil {
		t.Errorf("Failed to search expenses: %v", err)
	}

	if len(expenses) != 3 {
		t.Errorf("Expected 3 matching expenses, got %d", len(expenses))
	}

	for _, exp := range expenses {
		if exp.Description != "Test expense 1" && exp.Description != "Test expense 2" &&
			exp.Description != "Test expense 3" {
			t.Errorf("Unexpected expense description: %s", exp.Description)
		}
	}
}

func TestGetExpense(t *testing.T) {
	database := setupTestDB(t)

	now := time.Now()
	expenses := []*Expense{
		{
			Source:      "test_source",
			Amount:      1500,
			Description: "Test expense for retrieval",
			Type:        ChargeType,
			Date:        now,
			Currency:    "USD",
		},
	}

	insertedIDs, err := InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	if insertedIDs == 0 {
		t.Fatal("No expenses were inserted")
	}

	expense, err := GetExpense(database, 1)
	if err != nil {
		t.Errorf("Failed to get expense: %v", err)
	}

	if expense.Source != "test_source" {
		t.Errorf("Expected source 'test_source', got '%s'", expense.Source)
	}
	if expense.Amount != 1500 {
		t.Errorf("Expected amount 1500, got %d", expense.Amount)
	}
	if expense.Description != "Test expense for retrieval" {
		t.Errorf("Expected description 'Test expense for retrieval', got '%s'", expense.Description)
	}
	if expense.Type != ChargeType {
		t.Errorf("Expected type ChargeType, got %v", expense.Type)
	}
	if expense.Currency != "USD" {
		t.Errorf("Expected currency 'USD', got '%s'", expense.Currency)
	}
}

func TestGetExpenseNotFound(t *testing.T) {
	database := setupTestDB(t)

	_, err := GetExpense(database, 999)
	if err == nil {
		t.Error("Expected error when getting non-existent expense, but got none")
	}
}

func TestUpdateExpense(t *testing.T) {
	database := setupTestDB(t)

	categoryID, err := CreateCategory(database, "Test Category", "test")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	now := time.Now()
	originalExpense := []*Expense{
		{
			Source:      "original_source",
			Amount:      1000,
			Description: "Original description",
			Type:        ChargeType,
			Date:        now,
			Currency:    "EUR",
		},
	}

	_, err = InsertExpenses(database, originalExpense)
	if err != nil {
		t.Fatalf("Failed to insert original expense: %v", err)
	}

	updatedDate := now.AddDate(0, 0, 1)
	updatedExpense := &Expense{
		ID:          1,
		Source:      "updated_source",
		Amount:      2000,
		Description: "Updated description",
		Type:        IncomeType,
		Date:        updatedDate,
		Currency:    "USD",
		CategoryID:  sql.NullInt64{Int64: categoryID, Valid: true},
	}

	updated, err := UpdateExpense(database, updatedExpense)
	if err != nil {
		t.Errorf("Failed to update expense: %v", err)
	}

	if updated != 1 {
		t.Errorf("updated should be 1 got: %d", updated)
	}

	retrievedExpense, err := GetExpense(database, 1)
	if err != nil {
		t.Fatalf("Failed to retrieve updated expense: %v", err)
	}

	if retrievedExpense.Source != "updated_source" {
		t.Errorf("Expected source 'updated_source', got '%s'", retrievedExpense.Source)
	}
	if retrievedExpense.Amount != 2000 {
		t.Errorf("Expected amount 2000, got %d", retrievedExpense.Amount)
	}
	if retrievedExpense.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", retrievedExpense.Description)
	}
	if retrievedExpense.Type != IncomeType {
		t.Errorf("Expected type IncomeType, got %v", retrievedExpense.Type)
	}
	if retrievedExpense.Currency != "USD" {
		t.Errorf("Expected currency 'USD', got '%s'", retrievedExpense.Currency)
	}
	if !retrievedExpense.CategoryID.Valid || retrievedExpense.CategoryID.Int64 != categoryID {
		t.Errorf("Expected category ID %d, got %v", categoryID, retrievedExpense.CategoryID)
	}

	if retrievedExpense.Date.Unix() != updatedDate.Unix() {
		t.Errorf("Expected date %v, got %v", updatedDate, retrievedExpense.Date)
	}
}

func TestUpdateExpenseNonExistent(t *testing.T) {
	database := setupTestDB(t)

	nonExistentExpense := &Expense{
		ID:          999,
		Source:      "test",
		Amount:      1000,
		Description: "Test",
		Type:        ChargeType,
		Date:        time.Now(),
		Currency:    "USD",
	}

	updated, err := UpdateExpense(database, nonExistentExpense)

	if err != nil {
		t.Errorf("Unexpected error updating non-existent expense: %v", err)
	}

	if updated != 0 {
		t.Errorf("Unexpected updated value for non existing expense")
	}
}

func TestDeleteExpense(t *testing.T) {
	database := setupTestDB(t)

	now := time.Now()
	expenses := []*Expense{
		{
			Source:      "test1",
			Amount:      1000,
			Description: "Test expense 1",
			Type:        ChargeType,
			Date:        now,
			Currency:    "USD",
		},
		{
			Source:      "test2",
			Amount:      2000,
			Description: "Test expense 2",
			Type:        IncomeType,
			Date:        now,
			Currency:    "EUR",
		},
	}

	_, err := InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	allExpenses, err := GetExpenses(database)
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}
	if len(allExpenses) != 2 {
		t.Fatalf("Expected 2 expenses, got %d", len(allExpenses))
	}

	deleted, err := DeleteExpense(database, 1)
	if err != nil {
		t.Errorf("Failed to delete expense: %v", err)
	}

	if deleted != 1 {
		t.Errorf("deleted should be 1 got: %d", deleted)
	}

	allExpenses, err = GetExpenses(database)
	if err != nil {
		t.Fatalf("Failed to get expenses after deletion: %v", err)
	}
	if len(allExpenses) != 1 {
		t.Errorf("Expected 1 expense after deletion, got %d", len(allExpenses))
	}

	if allExpenses[0].Description != "Test expense 2" {
		t.Errorf("Expected remaining expense 'Test expense 2', got '%s'", allExpenses[0].Description)
	}

	_, err = GetExpense(database, 1)
	if err == nil {
		t.Error("Expected error when getting deleted expense, but got none")
	}
}

func TestDeleteExpenseNonExistent(t *testing.T) {
	database := setupTestDB(t)

	deleted, err := DeleteExpense(database, 999)

	if err != nil {
		t.Errorf("Unexpected error deleting non-existent expense: %v", err)
	}

	if deleted != 0 {
		t.Errorf("deleted should be 0 got: %d", deleted)
	}
}

func TestDeleteExpenseWithCategory(t *testing.T) {
	database := setupTestDB(t)

	categoryID, err := CreateCategory(database, "Test Category", "test")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	now := time.Now()
	expense := []*Expense{
		{
			Source:      "test",
			Amount:      1000,
			Description: "Test expense with category",
			Type:        ChargeType,
			Date:        now,
			Currency:    "USD",
			CategoryID:  sql.NullInt64{Int64: categoryID, Valid: true},
		},
	}

	_, err = InsertExpenses(database, expense)
	if err != nil {
		t.Fatalf("Failed to insert expense with category: %v", err)
	}

	_, err = DeleteExpense(database, 1)
	if err != nil {
		t.Errorf("Failed to delete expense with category: %v", err)
	}

	allExpenses, err := GetExpenses(database)
	if err != nil {
		t.Fatalf("Failed to get expenses after deletion: %v", err)
	}
	if len(allExpenses) != 0 {
		t.Errorf("Expected 0 expenses after deletion, got %d", len(allExpenses))
	}

	categories, err := GetCategories(database)
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}
	if len(categories) != 1 {
		t.Errorf("Expected 1 category to remain, got %d", len(categories))
	}
}
