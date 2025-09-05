package router

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestExpensesHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
	}

	for _, c := range categories {
		_, err := db.CreateCategory(database, c.Name, c.Pattern)
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	matcher := category.NewMatcher(categories)

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

	_, err := db.InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler, _ := New(database, matcher, logger)

	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

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

	if len(years) != 1 {
		t.Errorf("Expected 1 year, got %d", len(years))
	}
	if years[0] != now.Year() {
		t.Errorf("Expected year %d, got %d", now.Year(), years[0])
	}

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

func TestExpenseHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	categoryID, err := db.CreateCategory(database, "Test Category", "test")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	now := time.Now()
	expenses := []*db.Expense{
		{
			Source:      "Test Source",
			Date:        now,
			Description: "Test expense for edit",
			Amount:      -123456,
			Type:        db.ChargeType,
			Currency:    "USD",
			CategoryID:  sql.NullInt64{Int64: categoryID, Valid: true},
		},
	}

	_, err = db.InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	matcher := category.NewMatcher([]db.Category{})
	handler, _ := New(database, matcher, logger)

	req := httptest.NewRequest(http.MethodGet, "/expense/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test expense for edit") {
		t.Error("Response should contain expense description")
	}
}

func TestExpenseHandlerNotFound(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	matcher := category.NewMatcher([]db.Category{})
	handler, _ := New(database, matcher, logger)

	req := httptest.NewRequest(http.MethodGet, "/expense/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status Ok; got %v", resp.Status)
	}
}

func TestExpenseHandlerInvalidID(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	matcher := category.NewMatcher([]db.Category{})
	handler, _ := New(database, matcher, logger)

	req := httptest.NewRequest(http.MethodGet, "/expense/invalid", nil)
	req.SetPathValue("id", "invalid")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status Ok; got %v", resp.Status)
	}
}

func TestUpdateExpenseHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	categoryID, err := db.CreateCategory(database, "Updated Category", "updated")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	now := time.Now()
	expenses := []*db.Expense{
		{
			Source:      "Original Source",
			Date:        now,
			Description: "Original description",
			Amount:      -100000,
			Type:        db.ChargeType,
			Currency:    "EUR",
		},
	}

	_, err = db.InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	matcher := category.NewMatcher([]db.Category{})
	handler, _ := New(database, matcher, logger)

	formData := url.Values{}
	formData.Set("source", "Updated Source")
	formData.Set("description", "Updated description")
	formData.Set("amount", "20.50")
	formData.Set("currency", "USD")
	formData.Set("date", now.AddDate(0, 0, 1).Format("2006-01-02"))
	formData.Set("type", "1")
	formData.Set("category_id", strconv.FormatInt(categoryID, 10))

	req := httptest.NewRequest(http.MethodPut, "/expense/1", strings.NewReader(formData.Encode()))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	updatedExpense, err := db.GetExpense(database, 1)
	if err != nil {
		t.Fatalf("Failed to retrieve updated expense: %v", err)
	}

	if updatedExpense.Source != "Updated Source" {
		t.Errorf("Expected source 'Updated Source', got '%s'", updatedExpense.Source)
	}
	if updatedExpense.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", updatedExpense.Description)
	}
	if updatedExpense.Amount != 2050 {
		t.Errorf("Expected amount 2050, got %d", updatedExpense.Amount)
	}
	if updatedExpense.Currency != "USD" {
		t.Errorf("Expected currency 'USD', got '%s'", updatedExpense.Currency)
	}
	if updatedExpense.Type != db.IncomeType {
		t.Errorf("Expected type IncomeType, got %v", updatedExpense.Type)
	}
	if !updatedExpense.CategoryID.Valid || updatedExpense.CategoryID.Int64 != categoryID {
		t.Errorf("Expected category ID %d, got %v", categoryID, updatedExpense.CategoryID)
	}
}

func TestUpdateExpenseHandlerValidationErrors(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	now := time.Now()
	expenses := []*db.Expense{
		{
			Source:      "Test Source",
			Date:        now,
			Description: "Test expense",
			Amount:      -100000,
			Type:        db.ChargeType,
			Currency:    "USD",
		},
	}

	_, err := db.InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	matcher := category.NewMatcher([]db.Category{})
	handler, _ := New(database, matcher, logger)

	tests := []struct {
		name        string
		formData    map[string]string
		expectError bool
	}{
		{
			name: "missing source",
			formData: map[string]string{
				"description": "Test",
				"amount":      "10.00",
				"currency":    "USD",
				"date":        now.Format("2006-01-02"),
				"type":        "0",
			},
			expectError: true,
		},
		{
			name: "missing description",
			formData: map[string]string{
				"source":   "Test Source",
				"amount":   "10.00",
				"currency": "USD",
				"date":     now.Format("2006-01-02"),
				"type":     "0",
			},
			expectError: true,
		},
		{
			name: "invalid amount",
			formData: map[string]string{
				"source":      "Test Source",
				"description": "Test",
				"amount":      "invalid",
				"currency":    "USD",
				"date":        now.Format("2006-01-02"),
				"type":        "0",
			},
			expectError: true,
		},
		{
			name: "invalid date",
			formData: map[string]string{
				"source":      "Test Source",
				"description": "Test",
				"amount":      "10.00",
				"currency":    "USD",
				"date":        "invalid-date",
				"type":        "0",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formData := url.Values{}
			for key, value := range tt.formData {
				formData.Set(key, value)
			}

			req := httptest.NewRequest(http.MethodPut, "/expense/1", strings.NewReader(formData.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK for validation error handling; got %v", resp.Status)
			}

			if w.Header().Get("Hx-Redirect") != "" {
				t.Error("Should not redirect when there are validation errors")
			}

			body := w.Body.String()
			if !strings.Contains(body, "error") && !strings.Contains(body, "required") {
				t.Error("Response should contain error information")
			}
		})
	}
}

func TestDeleteExpenseHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	now := time.Now()
	expenses := []*db.Expense{
		{
			Source:      "Test Source 1",
			Date:        now,
			Description: "Test expense 1",
			Amount:      -100000,
			Type:        db.ChargeType,
			Currency:    "USD",
		},
		{
			Source:      "Test Source 2",
			Date:        now,
			Description: "Test expense 2",
			Amount:      -200000,
			Type:        db.ChargeType,
			Currency:    "USD",
		},
	}

	_, err := db.InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	matcher := category.NewMatcher([]db.Category{})
	handler, _ := New(database, matcher, logger)

	req := httptest.NewRequest(http.MethodDelete, "/expense/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	if w.Header().Get("Hx-Redirect") != "/expenses" {
		t.Error("Expected Hx-Redirect header to /expenses")
	}

	allExpenses, err := db.GetExpenses(database)
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(allExpenses) != 1 {
		t.Errorf("Expected 1 expense after deletion, got %d", len(allExpenses))
	}

	if allExpenses[0].Description != "Test expense 2" {
		t.Errorf("Expected remaining expense 'Test expense 2', got '%s'", allExpenses[0].Description)
	}

	_, err = db.GetExpense(database, 1)
	if err == nil {
		t.Error("Expected error when getting deleted expense")
	}
}

func TestDeleteExpenseHandlerNotFound(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	matcher := category.NewMatcher([]db.Category{})
	handler, _ := New(database, matcher, logger)

	req := httptest.NewRequest(http.MethodDelete, "/expense/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status Ok; got %v", resp.Status)
	}
}

func TestExpenseHandlersIntegration(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	categoryID, err := db.CreateCategory(database, "Integration Category", "integration")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	matcher := category.NewMatcher([]db.Category{
		{ID: categoryID, Name: "Integration Category", Pattern: "integration"},
	})
	handler, _ := New(database, matcher, logger)

	now := time.Now()
	expenses := []*db.Expense{
		{
			Source:      "Integration Source",
			Date:        now,
			Description: "Integration test expense",
			Amount:      -500000,
			Type:        db.ChargeType,
			Currency:    "EUR",
		},
	}

	_, err = db.InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/expense/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("GET /expense/1 failed with status %v", w.Result().Status)
	}

	formData := url.Values{}
	formData.Set("source", "Updated Integration Source")
	formData.Set("description", "Updated integration test expense")
	formData.Set("amount", "75.25")
	formData.Set("currency", "USD")
	formData.Set("date", now.Format("2006-01-02"))
	formData.Set("type", "1")
	formData.Set("category_id", strconv.FormatInt(categoryID, 10))

	req = httptest.NewRequest(http.MethodPut, "/expense/1", strings.NewReader(formData.Encode()))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("PUT /expense/1 failed with status %v", w.Result().Status)
	}

	updatedExpense, err := db.GetExpense(database, 1)
	if err != nil {
		t.Fatalf("Failed to get updated expense: %v", err)
	}

	if updatedExpense.Source != "Updated Integration Source" {
		t.Errorf("Source not updated correctly")
	}
	if updatedExpense.Amount != 7525 {
		t.Errorf("Amount not updated correctly: got %d, expected 7525", updatedExpense.Amount)
	}
	if updatedExpense.Type != db.IncomeType {
		t.Errorf("Type not updated correctly")
	}

	req = httptest.NewRequest(http.MethodGet, "/expense/1/delete", nil)
	req.SetPathValue("id", "1")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("GET /expense/1/delete failed with status %v", w.Result().Status)
	}

	req = httptest.NewRequest(http.MethodDelete, "/expense/1", nil)
	req.SetPathValue("id", "1")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("DELETE /expense/1 failed with status %v", w.Result().Status)
	}

	allExpenses, err := db.GetExpenses(database)
	if err != nil {
		t.Fatalf("Failed to get expenses after deletion: %v", err)
	}

	if len(allExpenses) != 0 {
		t.Errorf("Expected 0 expenses after deletion, got %d", len(allExpenses))
	}
}

func TestExpenseSearchHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
	}

	for _, c := range categories {
		_, err := db.CreateCategory(database, c.Name, c.Pattern)
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	matcher := category.NewMatcher(categories)

	// Create test expenses
	expenses := []*db.Expense{
		{
			Source:      "Test Source",
			Date:        time.Now(),
			Description: "Restaurant bill",
			Amount:      -123456,
			Type:        db.ChargeType,
			Currency:    "USD",
			CategoryID:  sql.NullInt64{Int64: int64(1), Valid: true},
		},
	}

	_, err := db.InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	// Create router
	handler, _ := New(database, matcher, logger)

	// Create test request
	body := strings.NewReader("keyword=restaurant")
	req := httptest.NewRequest(http.MethodPost, "/expense/search", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Serve request
	handler.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}
}
