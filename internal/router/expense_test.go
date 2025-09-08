package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestExpensesHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
		storage.NewCategory(2, "Transport", "uber|taxi|transit"),
	}
	categoryIDs := make([]int64, 2)
	for i, c := range categories {
		id, err := s.CreateCategory(c.Name(), c.Pattern())
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
		categoryIDs[i] = id
	}

	matcher := matcher.New(categories)

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(
			0,
			"Test Source",
			"Restaurant bill",
			"USD",
			-123456,
			now,
			storage.ChargeType,
			&categoryIDs[0],
		),
		storage.NewExpense(0, "Test Source", "Uber ride", "USD", -50000, now, storage.ChargeType, &categoryIDs[1]),
	}

	_, err := s.InsertExpenses(expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler, _ := New(s, matcher, logger)

	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "expenses", resp.Body)
}

func TestExpensesGroupByYearAndMonth(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	cat1 := int64(1)
	cat2 := int64(2)
	now := time.Now()

	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, storage.ChargeType, &cat1),
		storage.NewExpense(0, "Test Source", "Uber ride", "USD", -50000, now, storage.ChargeType, &cat2),
	}

	groupedExpenses, years, err := expensesGroupByYearAndMonth(expenses, s)

	if err != nil {
		t.Fatalf("Got error grouping expenses: %s", err.Error())
	}

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
		if expense.CategoryID() == cat1 {
			foundFood = true
		}
		if expense.CategoryID() == cat2 {
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
	s := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory("Test Category", "test")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(
			0,
			"Test Source",
			"Test expense for edit",
			"USD",
			-123456,
			now,
			storage.ChargeType,
			&categoryID,
		),
	}

	_, err = s.InsertExpenses(expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	matcher := matcher.New([]storage.Category{})
	handler, _ := New(s, matcher, logger)

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

	ensureNoErrorInTemplateResponse(t, "expense", resp.Body)
}

func TestExpenseHandlerNotFound(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	matcher := matcher.New([]storage.Category{})
	handler, _ := New(s, matcher, logger)

	req := httptest.NewRequest(http.MethodGet, "/expense/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status Ok; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "expense not found", resp.Body)
}

func TestExpenseHandlerInvalidID(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	matcher := matcher.New([]storage.Category{})
	handler, _ := New(s, matcher, logger)

	req := httptest.NewRequest(http.MethodGet, "/expense/invalid", nil)
	req.SetPathValue("id", "invalid")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status Ok; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "expense invalid ID", resp.Body)
}

func TestUpdateExpenseHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory("Updated Category", "updated")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Original Source", "Original description", "EUR", -100000, now, storage.ChargeType, nil),
	}

	_, err = s.InsertExpenses(expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	matcher := matcher.New([]storage.Category{})
	handler, _ := New(s, matcher, logger)

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

	ensureNoErrorInTemplateResponse(t, "update expense", resp.Body)

	updatedExpense, err := s.GetExpenseByID(1)
	if err != nil {
		t.Fatalf("Failed to retrieve updated expense: %v", err)
	}

	if updatedExpense.Source() != "Updated Source" {
		t.Errorf("Expected source 'Updated Source', got '%s'", updatedExpense.Source())
	}
	if updatedExpense.Description() != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", updatedExpense.Description())
	}
	if updatedExpense.Amount() != 2050 {
		t.Errorf("Expected amount 2050, got %d", updatedExpense.Amount())
	}
	if updatedExpense.Currency() != "USD" {
		t.Errorf("Expected currency 'USD', got '%s'", updatedExpense.Currency())
	}
	if updatedExpense.Type() != storage.IncomeType {
		t.Errorf("Expected type IncomeType, got %v", updatedExpense.Type())
	}
	if *updatedExpense.CategoryID() != categoryID {
		t.Errorf("Expected category ID %d, got %d", categoryID, *updatedExpense.CategoryID())
	}
}

func TestUpdateExpenseHandlerValidationErrors(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Test expense", "USD", -100000, now, storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	matcher := matcher.New([]storage.Category{})
	handler, _ := New(s, matcher, logger)

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

			ensureNoErrorInTemplateResponse(t, fmt.Sprintf("update expense: %s", tt.name), resp.Body)

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
	s := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source 1", "Test expense 1", "USD", -100000, now, storage.ChargeType, nil),
		storage.NewExpense(0, "Test Source 2", "Test expense 2", "USD", -200000, now, storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	matcher := matcher.New([]storage.Category{})
	handler, _ := New(s, matcher, logger)

	req := httptest.NewRequest(http.MethodDelete, "/expense/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "delete expense", resp.Body)

	if w.Header().Get("Hx-Redirect") != "/expenses" {
		t.Error("Expected Hx-Redirect header to /expenses")
	}

	allExpenses, err := s.GetExpenses()
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(allExpenses) != 1 {
		t.Errorf("Expected 1 expense after deletion, got %d", len(allExpenses))
	}

	if allExpenses[0].Description() != "Test expense 2" {
		t.Errorf("Expected remaining expense 'Test expense 2', got '%s'", allExpenses[0].Description())
	}

	_, err = s.GetExpenseByID(1)
	if err == nil {
		t.Error("Expected error when getting deleted expense")
	}
}

func TestDeleteExpenseHandlerNotFound(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	matcher := matcher.New([]storage.Category{})
	handler, _ := New(s, matcher, logger)

	req := httptest.NewRequest(http.MethodDelete, "/expense/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status Ok; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "delete expense not found", resp.Body)
}

func TestExpenseHandlersIntegration(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory("Integration Category", "integration")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	matcher := matcher.New([]storage.Category{
		storage.NewCategory(categoryID, "Integration Category", "integration"),
	})
	handler, _ := New(s, matcher, logger)

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(
			0,
			"Integration Source",
			"Integration test expense",
			"EUR",
			-500000,
			now,
			storage.ChargeType,
			&categoryID,
		),
	}

	_, err = s.InsertExpenses(expenses)
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

	updatedExpense, err := s.GetExpenseByID(1)
	if err != nil {
		t.Fatalf("Failed to get updated expense: %v", err)
	}

	if updatedExpense.Source() != "Updated Integration Source" {
		t.Errorf("Source not updated correctly")
	}
	if updatedExpense.Amount() != 7525 {
		t.Errorf("Amount not updated correctly: got %d, expected 7525", updatedExpense.Amount())
	}
	if updatedExpense.Type() != storage.IncomeType {
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

	allExpenses, err := s.GetExpenses()
	if err != nil {
		t.Fatalf("Failed to get expenses after deletion: %v", err)
	}

	if len(allExpenses) != 0 {
		t.Errorf("Expected 0 expenses after deletion, got %d", len(allExpenses))
	}
}

func TestExpenseSearchHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	// Create test categories
	categoryID, err := s.CreateCategory("Food", "restaurant|food|grocery")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	categories := []storage.Category{
		storage.NewCategory(categoryID, "Food", "restaurant|food|grocery"),
	}

	matcher := matcher.New(categories)

	// Create test expenses
	expenses := []storage.Expense{
		storage.NewExpense(
			0,
			"Test Source",
			"Restaurant bill",
			"USD",
			-123456,
			time.Now(),
			storage.ChargeType,
			&categoryID,
		),
	}

	_, err = s.InsertExpenses(expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	// Create router
	handler, _ := New(s, matcher, logger)

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

	ensureNoErrorInTemplateResponse(t, "search expenses", resp.Body)
}
