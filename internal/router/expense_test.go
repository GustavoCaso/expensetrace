package router

import (
	"context"

	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestExpensesHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery", 0),
		storage.NewCategory(2, "Transport", "uber|taxi|transit", 0),
	}
	categoryIDs := make([]int64, 2)
	for i, c := range categories {
		id, err := s.CreateCategory(context.Background(), user.ID(), c.Name(), c.Pattern(), 0)
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
		categoryIDs[i] = id
	}

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

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
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
	s, user := testutil.SetupTestStorage(t, logger)

	cat1 := int64(1)
	cat2 := int64(2)
	now := time.Now()

	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, storage.ChargeType, &cat1),
		storage.NewExpense(0, "Test Source", "Uber ride", "USD", -50000, now, storage.ChargeType, &cat2),
	}

	groupedExpenses, years, err := expensesGroupByYearAndMonth(context.Background(), user.ID(), expenses, s)

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
	s, user := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory(context.Background(), user.ID(), "Test Category", "test", 0)
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

	_, err = s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodGet, "/expense/1", nil)
	req.SetPathValue("id", "1")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
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
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodGet, "/expense/999", nil)
	req.SetPathValue("id", "999")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
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
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodGet, "/expense/invalid", nil)
	req.SetPathValue("id", "invalid")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
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
	s, user := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory(context.Background(), user.ID(), "Updated Category", "updated", 0)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Original Source", "Original description", "EUR", -100000, now, storage.ChargeType, nil),
	}

	_, err = s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	handler, _ := New(s, logger)

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
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "update expense", resp.Body)

	updatedExpense, err := s.GetExpenseByID(context.Background(), user.ID(), 1)
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
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Test expense", "USD", -100000, now, storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	handler, _ := New(s, logger)

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
			testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
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
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source 1", "Test expense 1", "USD", -100000, now, storage.ChargeType, nil),
		storage.NewExpense(0, "Test Source 2", "Test expense 2", "USD", -200000, now, storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodDelete, "/expense/1", nil)
	req.SetPathValue("id", "1")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "delete expense", resp.Body)

	allExpenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(allExpenses) != 1 {
		t.Errorf("Expected 1 expense after deletion, got %d", len(allExpenses))
	}

	if allExpenses[0].Description() != "Test expense 2" {
		t.Errorf("Expected remaining expense 'Test expense 2', got '%s'", allExpenses[0].Description())
	}

	_, err = s.GetExpenseByID(context.Background(), user.ID(), 1)
	if err == nil {
		t.Error("Expected error when getting deleted expense")
	}
}

func TestDeleteExpenseHandlerNotFound(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodDelete, "/expense/999", nil)
	req.SetPathValue("id", "999")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
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
	s, user := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory(context.Background(), user.ID(), "Integration Category", "integration", 0)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	handler, _ := New(s, logger)

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

	_, err = s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expense: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/expense/1", nil)
	req.SetPathValue("id", "1")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
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
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("PUT /expense/1 failed with status %v", w.Result().Status)
	}

	updatedExpense, err := s.GetExpenseByID(context.Background(), user.ID(), 1)
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

	req = httptest.NewRequest(http.MethodDelete, "/expense/1", nil)
	req.SetPathValue("id", "1")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("DELETE /expense/1 failed with status %v", w.Result().Status)
	}

	allExpenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses after deletion: %v", err)
	}

	if len(allExpenses) != 0 {
		t.Errorf("Expected 0 expenses after deletion, got %d", len(allExpenses))
	}
}

func TestCreateExpenseHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory(context.Background(), user.ID(), "Test Category", "test", 0)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	handler, _ := New(s, logger)

	now := time.Now()
	formData := url.Values{}
	formData.Set("source", "Test Source")
	formData.Set("description", "Test expense creation")
	formData.Set("amount", "25.50")
	formData.Set("currency", "USD")
	formData.Set("date", now.Format("2006-01-02"))
	formData.Set("type", "0")
	formData.Set("category_id", strconv.FormatInt(categoryID, 10))

	req := httptest.NewRequest(http.MethodPost, "/expense", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "create expense", resp.Body)

	body := w.Body.String()
	if !strings.Contains(body, "Expense Created") {
		t.Error("Response should contain success banner")
	}

	allExpenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(allExpenses) != 1 {
		t.Errorf("Expected 1 expense after creation, got %d", len(allExpenses))
	}

	expense := allExpenses[0]
	if expense.Source() != "Test Source" {
		t.Errorf("Expected source 'Test Source', got '%s'", expense.Source())
	}
	if expense.Description() != "Test expense creation" {
		t.Errorf("Expected description 'Test expense creation', got '%s'", expense.Description())
	}
	if expense.Amount() != -2550 {
		t.Errorf("Expected amount -2550 (cents), got %d", expense.Amount())
	}
	if expense.Currency() != "USD" {
		t.Errorf("Expected currency 'USD', got '%s'", expense.Currency())
	}
	if expense.Type() != storage.ChargeType {
		t.Errorf("Expected type ChargeType, got %v", expense.Type())
	}
	if *expense.CategoryID() != categoryID {
		t.Errorf("Expected category ID %d, got %d", categoryID, *expense.CategoryID())
	}
}

func TestCreateExpenseHandlerNilCategory(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	now := time.Now()
	formData := url.Values{}
	formData.Set("source", "Test Source")
	formData.Set("description", "Test expense without category")
	formData.Set("amount", "10.00")
	formData.Set("currency", "EUR")
	formData.Set("date", now.Format("2006-01-02"))
	formData.Set("type", "1")
	formData.Set("category_id", "")

	req := httptest.NewRequest(http.MethodPost, "/expense", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Expense Created") {
		t.Errorf("Expected success banner, got: %s", body)
	}

	allExpenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(allExpenses) != 1 {
		t.Errorf("Expected 1 expense after creation, got %d", len(allExpenses))
		return
	}

	expense := allExpenses[0]
	if expense.CategoryID() != nil {
		t.Errorf("Expected nil category ID, got %v", expense.CategoryID())
	}
	if expense.Type() != storage.IncomeType {
		t.Errorf("Expected type IncomeType, got %v", expense.Type())
	}
}

func TestCreateExpenseHandlerAmountSigning(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	now := time.Now()

	tests := []struct {
		name           string
		amount         string
		expenseType    string
		expectedAmount int64
		description    string
	}{
		{
			name:           "positive amount expense becomes negative",
			amount:         "10.50",
			expenseType:    "0", // ChargeType
			expectedAmount: -1050,
			description:    "Positive expense amount",
		},
		{
			name:           "negative amount expense stays negative",
			amount:         "-10.50",
			expenseType:    "0", // ChargeType
			expectedAmount: -1050,
			description:    "Negative expense amount",
		},
		{
			name:           "positive amount income stays positive",
			amount:         "15.75",
			expenseType:    "1", // IncomeType
			expectedAmount: 1575,
			description:    "Positive income amount",
		},
		{
			name:           "negative amount income becomes positive",
			amount:         "-15.75",
			expenseType:    "1", // IncomeType
			expectedAmount: 1575,
			description:    "Negative income amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formData := url.Values{}
			formData.Set("source", "Test Source")
			formData.Set("description", tt.description)
			formData.Set("amount", tt.amount)
			formData.Set("currency", "USD")
			formData.Set("date", now.Format("2006-01-02"))
			formData.Set("type", tt.expenseType)
			formData.Set("category_id", "")

			req := httptest.NewRequest(http.MethodPost, "/expense", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK; got %v", resp.Status)
			}

			body := w.Body.String()
			if !strings.Contains(body, "Expense Created") {
				t.Errorf("Expected success banner, got: %s", body)
			}

			allExpenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
			if err != nil {
				t.Fatalf("Failed to get expenses: %v", err)
			}

			if len(allExpenses) == 0 {
				t.Fatal("Expected expense to be created")
			}

			expense := allExpenses[len(allExpenses)-1] // Get the last created expense
			if expense.Amount() != tt.expectedAmount {
				t.Errorf("Expected amount %d, got %d", tt.expectedAmount, expense.Amount())
			}
		})
	}
}

func TestCreateExpenseHandlerValidationErrors(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	now := time.Now()

	tests := []struct {
		name          string
		formData      map[string]string
		expectedError string
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
			expectedError: "Source is required",
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
			expectedError: "Description is required",
		},
		{
			name: "missing currency",
			formData: map[string]string{
				"source":      "Test Source",
				"description": "Test",
				"amount":      "10.00",
				"date":        now.Format("2006-01-02"),
				"type":        "0",
			},
			expectedError: "Currency is required",
		},
		{
			name: "missing amount",
			formData: map[string]string{
				"source":      "Test Source",
				"description": "Test",
				"currency":    "USD",
				"date":        now.Format("2006-01-02"),
				"type":        "0",
			},
			expectedError: "Amount is required",
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
			expectedError: "Invalid amount format",
		},
		{
			name: "missing date",
			formData: map[string]string{
				"source":      "Test Source",
				"description": "Test",
				"amount":      "10.00",
				"currency":    "USD",
				"type":        "0",
			},
			expectedError: "Date is required",
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
			expectedError: "Invalid date format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formData := url.Values{}
			for key, value := range tt.formData {
				formData.Set(key, value)
			}

			req := httptest.NewRequest(http.MethodPost, "/expense", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK for validation error handling; got %v", resp.Status)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.expectedError) {
				t.Errorf("Expected error message '%s' not found in response", tt.expectedError)
			}

			if strings.Contains(body, "Expense Created") {
				t.Error("Should not show success banner when there are validation errors")
			}

			allExpenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
			if err != nil {
				t.Fatalf("Failed to get expenses: %v", err)
			}

			if len(allExpenses) != 0 {
				t.Errorf("Expected 0 expenses after validation error, got %d", len(allExpenses))
			}
		})
	}
}
func TestCreateExpenseHandlerFormParseError(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodPost, "/expense", strings.NewReader("%zzzzz"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "create expense form parse error", resp.Body)
}

func TestExpensesHandlerWithFilters(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	// Insert test expenses with varied data for comprehensive testing
	expenses := []storage.Expense{
		storage.NewExpense(0, "visa", "morning coffee", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "mastercard", "lunch", "USD", -1200, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "visa", "afternoon coffee", "USD", -450, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "visa", "grocery shopping", "USD", -8000, time.Date(2024, 2, 5, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "employer", "salary", "USD", 500000, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), storage.IncomeType, nil),
	}
	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("failed to insert expenses: %v", err)
	}

	handler, _ := New(s, logger)

	tests := []struct {
		name       string
		url        string
		wantStatus int
		checkBody  func(t *testing.T, body string)
	}{
		{
			name:       "no filters returns all expenses",
			url:        "/expenses",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "morning coffee") {
					t.Error("expected to find 'morning coffee'")
				}
				if !strings.Contains(body, "lunch") {
					t.Error("expected to find 'lunch'")
				}
				if !strings.Contains(body, "salary") {
					t.Error("expected to find 'salary'")
				}
			},
		},
		{
			name:       "filter by description",
			url:        "/expenses?description=coffee",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "morning coffee") {
					t.Error("expected to find 'morning coffee'")
				}
				if !strings.Contains(body, "afternoon coffee") {
					t.Error("expected to find 'afternoon coffee'")
				}
				if strings.Contains(body, "lunch") {
					t.Error("did not expect to find 'lunch'")
				}
			},
		},
		{
			name:       "filter by source",
			url:        "/expenses?source=visa",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "morning coffee") {
					t.Error("expected to find 'morning coffee'")
				}
				if strings.Contains(body, "lunch") {
					t.Error("did not expect to find 'lunch'")
				}
			},
		},
		{
			name:       "filter by amount range",
			url:        "/expenses?amount_min=-12.00&amount_max=-4.00",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				// Amounts stored as negative for expenses
				// morning coffee: -500 cents (-$5.00), should be included
				// lunch: -1200 cents (-$12.00), should be included
				// afternoon coffee: -450 cents (-$4.50), should be included
				// grocery: -8000 cents (-$80.00), should NOT be included
				if !strings.Contains(body, "morning coffee") {
					t.Error("expected to find 'morning coffee' (-5.00)")
				}
				if !strings.Contains(body, "lunch") {
					t.Error("expected to find 'lunch' (-12.00)")
				}
				if strings.Contains(body, "grocery shopping") {
					t.Error("did not expect to find 'grocery shopping' (-80.00)")
				}
			},
		},
		{
			name:       "filter by date range",
			url:        "/expenses?date_from=2024-01-15&date_to=2024-01-17",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "morning coffee") {
					t.Error("expected to find 'morning coffee' (Jan 15)")
				}
				if !strings.Contains(body, "lunch") {
					t.Error("expected to find 'lunch' (Jan 16)")
				}
				if !strings.Contains(body, "afternoon coffee") {
					t.Error("expected to find 'afternoon coffee' (Jan 17)")
				}
				if strings.Contains(body, "grocery shopping") {
					t.Error("did not expect to find 'grocery shopping' (Feb 5)")
				}
			},
		},
		{
			name:       "combined filters - description and source",
			url:        "/expenses?description=coffee&source=visa",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "morning coffee") {
					t.Error("expected to find 'morning coffee' (visa + coffee)")
				}
				if !strings.Contains(body, "afternoon coffee") {
					t.Error("expected to find 'afternoon coffee' (visa + coffee)")
				}
				if strings.Contains(body, "lunch") {
					t.Error("did not expect to find 'lunch' (mastercard)")
				}
			},
		},
		{
			name:       "sort by amount descending",
			url:        "/expenses?sort=amount:desc",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				// Just verify the page renders with all expenses
				if !strings.Contains(body, "salary") {
					t.Error("expected to find 'salary' (highest amount)")
				}
			},
		},
		{
			name:       "sort by date ascending",
			url:        "/expenses?sort=date:asc",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				// Just verify the page renders with all expenses
				if !strings.Contains(body, "salary") {
					t.Error("expected to find 'salary' (earliest date)")
				}
			},
		},
		{
			name:       "invalid filter returns error",
			url:        "/expenses?amount_min=invalid",
			wantStatus: http.StatusOK, // Still renders page
			checkBody: func(t *testing.T, body string) {
				// The page should render with an error message about invalid filters
				// Looking for either "Invalid filters" or just "error" in the rendered page
				hasInvalidFilters := strings.Contains(body, "Invalid filters")
				hasError := strings.Contains(body, "error") || strings.Contains(body, "Error")
				if !hasInvalidFilters && !hasError {
					t.Errorf("expected error message about invalid filters, got body length: %d", len(body))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.String())
			}
		})
	}
}
