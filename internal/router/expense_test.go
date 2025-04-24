package router

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestExpensesHandler(t *testing.T) {
	database := testutil.SetupTestDB(t)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery", Type: db.ExpenseCategoryType},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit", Type: db.ExpenseCategoryType},
	}

	for _, c := range categories {
		_, err := db.CreateCategory(database, c.Name, c.Pattern, c.Type)
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	matcher := category.NewMatcher(categories)

	// Create test expenses
	now := time.Now()
	expenses := []*db.Expense{
		{
			Source:      "Test Source",
			Date:        now,
			Description: "Restaurant bill",
			Amount:      -123456,
			Type:        db.ChargeType,
			Currency:    "USD",
			CategoryID:  sql.NullInt64{Int64: int64(1), Valid: true},
		},
		{
			Source:      "Test Source",
			Date:        now,
			Description: "Uber ride",
			Amount:      -50000,
			Type:        db.ChargeType,
			Currency:    "USD",
			CategoryID:  sql.NullInt64{Int64: int64(2), Valid: true},
		},
	}

	err := db.InsertExpenses(database, expenses)
	if len(err) > 0 {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	// Create router
	handler, _ := New(database, matcher)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	w := httptest.NewRecorder()

	// Serve request
	handler.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}
}

func TestExpensesGroupByYearAndMonth(t *testing.T) {
	now := time.Now()
	expenses := []*db.Expense{
		{
			Source:      "Test Source",
			Date:        now,
			Description: "Restaurant bill",
			Amount:      -123456,
			Type:        db.ChargeType,
			Currency:    "USD",
			CategoryID:  sql.NullInt64{Int64: int64(1), Valid: true},
		},
		{
			Source:      "Test Source",
			Date:        now,
			Description: "Uber ride",
			Amount:      -50000,
			Type:        db.ChargeType,
			Currency:    "USD",
			CategoryID:  sql.NullInt64{Int64: int64(2), Valid: true},
		},
	}

	groupedExpenses, years := expensesGroupByYearAndMonth(expenses)

	// Check years
	if len(years) != 1 {
		t.Errorf("Expected 1 year, got %d", len(years))
	}
	if years[0] != now.Year() {
		t.Errorf("Expected year %d, got %d", now.Year(), years[0])
	}

	// Check grouped expenses
	yearExpenses, ok := groupedExpenses[now.Year()]
	if !ok {
		t.Error("Expected expenses for current year")
	}

	monthExpenses, ok := yearExpenses[now.Month().String()]
	if !ok {
		t.Error("Expected expenses for current month")
	}

	if len(monthExpenses) != 2 {
		t.Errorf("Expected 2 expenses, got %d", len(monthExpenses))
	}

	// Check expense details
	foundFood := false
	foundTransport := false
	for _, expense := range monthExpenses {
		if expense.CategoryID.Int64 == 1 {
			foundFood = true
		}
		if expense.CategoryID.Int64 == 2 {
			foundTransport = true
		}
	}

	if !foundFood {
		t.Error("Food expense not found")
	}
	if !foundTransport {
		t.Error("Transport expense not found")
	}
}
