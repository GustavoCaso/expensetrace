package db

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	err = ApplyMigrations(database)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Enable foreign key constraints
	_, err = database.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable PRAGMA: %v", err)
	}

	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("Failed to close test expenses database: %v", err)
		}
	})

	return database
}

func TestCreateExpenseTable(t *testing.T) {
	database := setupTestDB(t)

	// Verify table exists by trying to insert a record
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

	errs := InsertExpenses(database, expenses)
	if len(errs) > 0 {
		t.Errorf("Failed to insert expenses: %v", errs)
	}

	// Verify expenses were inserted
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

	// Create Category 1
	_, err := CreateCategory(database, "Test", "*")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	for _, exp := range testExpenses {
		_, err := database.Exec(
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
		// if exp.CategoryID == nil {
		// 	continue
		// }
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

	// Insert test expenses with different dates
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

	// Create Category 1
	_, err := CreateCategory(database, "Test", "*")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	for _, exp := range testExpenses {
		_, err := database.Exec(
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

	// Test getting expenses within date range
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

	// Create Category 1
	_, err := CreateCategory(database, "Test", "*")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	for _, exp := range testExpenses {
		_, err := database.Exec(
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

	// Create Category 1
	_, err := CreateCategory(database, "Test", "*")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	for _, exp := range testExpenses {
		_, err := database.Exec(
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
