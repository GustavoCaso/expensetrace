package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestCategoriesHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
		storage.NewCategory(2, "Transport", "uber|taxi|transit"),
	}
	matcher := matcher.New(categories)

	handler, _ := New(s, matcher, logger)

	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "categories", resp.Body)
}

func TestUncategorizedHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
	}
	matcher := matcher.New(categories)

	expenses := []storage.Expense{
		storage.NewExpense(
			0,
			"Test Source",
			"Uncategorized expense",
			"USD",
			-123456,
			time.Now(),
			storage.ChargeType,
			nil,
		),
	}

	_, err := s.InsertExpenses(expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler, _ := New(s, matcher, logger)

	req := httptest.NewRequest(http.MethodGet, "/uncategorized", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "uncategorized", resp.Body)
}

func TestUncategorizedSearchHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
	}
	matcher := matcher.New(categories)

	expenses := []storage.Expense{
		storage.NewExpense(
			0,
			"Test Source",
			"Coffee shop purchase",
			"USD",
			-500,
			time.Now(),
			storage.ChargeType,
			nil,
		),
		storage.NewExpense(
			0,
			"Test Source",
			"Hardware store",
			"USD",
			-1500,
			time.Now(),
			storage.ChargeType,
			nil,
		),
	}

	_, err := s.InsertExpenses(expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler, _ := New(s, matcher, logger)

	body := strings.NewReader("q=Coffee")
	req := httptest.NewRequest(http.MethodPost, "/category/uncategorized/search", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "uncategorized search", resp.Body)

	emptyBody := strings.NewReader("q=")
	emptyReq := httptest.NewRequest(http.MethodPost, "/category/uncategorized/search", emptyBody)
	emptyReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	emptyW := httptest.NewRecorder()

	handler.ServeHTTP(emptyW, emptyReq)

	emptyResp := emptyW.Result()
	if emptyResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK for empty query; got %v", emptyResp.Status)
	}

	responseBody := emptyW.Body.String()
	if !strings.Contains(responseBody, errSearchCriteria) {
		t.Error("Expected error message for empty search query")
	}

	invalidReq := httptest.NewRequest(
		http.MethodPost,
		"/category/uncategorized/search",
		strings.NewReader("invalid%form"),
	)
	invalidReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidW := httptest.NewRecorder()

	handler.ServeHTTP(invalidW, invalidReq)

	invalidResp := invalidW.Result()
	if invalidResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK for form parse error; got %v", invalidResp.Status)
	}
}

func TestCreateCategoryHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
	}
	matcher := matcher.New(categories)

	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "cinema", "USD", -123456, time.Now(), storage.ChargeType, nil),
	}

	_, expenseError := s.InsertExpenses(expenses)
	if expenseError != nil {
		t.Fatalf("Failed to insert test expenses: %v", expenseError)
	}

	handler, router := New(s, matcher, logger)

	oldMatcher := router.matcher
	oldSyncOnce := router.reportsOnce

	body := strings.NewReader("name=Entertainment&pattern=cinema|movie|theater&type=0")
	req := httptest.NewRequest(http.MethodPost, "/category", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "create category", resp.Body)

	categories, err := s.GetCategories()
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}

	found := false
	var categoryID int64
	for _, c := range categories {
		if c.Name() == "Entertainment" && c.Pattern() == "cinema|movie|theater" {
			found = true
			categoryID = c.ID()
			break
		}
	}

	if !found {
		t.Error("Category was not created")
	}

	expensesUpdated, err := s.SearchExpensesByDescription("cinema")

	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(expensesUpdated) > 1 {
		t.Fatalf("Failed more expenses than it should: %v", err)
	}

	if expensesUpdated[0].CategoryID() != nil && *expensesUpdated[0].CategoryID() != categoryID {
		t.Fatal("Expense did not update the category ID")
	}

	if oldMatcher == router.matcher {
		t.Error("Category matcher was not re-created")
	}

	if oldSyncOnce == router.reportsOnce {
		t.Error("Router cache was not reset")
	}
}

func TestUpdateHandler(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		updateMatcher bool
		assertion     func(t *testing.T, updatedCategory storage.Category, updatedExpenses []storage.Expense)
	}{
		{
			"modify pattern and set expense to NULL category",
			"pattern=test_pattern",
			true,
			func(t *testing.T, updatedCategory storage.Category, updatedExpenses []storage.Expense) {
				if updatedCategory.Pattern() != "test_pattern" {
					t.Fatalf(
						"Category was not updated properly. Expected pattern to be `test_pattern` but was %s",
						updatedCategory.Pattern(),
					)
				}

				for _, ex := range updatedExpenses {
					if ex.Description() == "cinema" {
						if ex.CategoryID() != nil {
							t.Fatalf(
								"Expense was not properly updated. Category ID must be NULL. Got %d",
								*ex.CategoryID(),
							)
						}
					}
				}
			},
		},
		{
			"modify pattern and update existing expenses",
			"pattern=restaurant|bars|cinema|gym",
			true,
			func(t *testing.T, updatedCategory storage.Category, updatedExpenses []storage.Expense) {
				if updatedCategory.Pattern() != "restaurant|bars|cinema|gym" {
					t.Fatalf(
						"Category was not updated properly. Expected pattern to be `restaurant|bars|cinema|gym` but was %s",
						updatedCategory.Pattern(),
					)
				}

				for _, ex := range updatedExpenses {
					if ex.CategoryID() != nil && *ex.CategoryID() != updatedCategory.ID() {
						t.Fatalf(
							"Expense %s was incoreectly updated. Category ID must be %d. Got %d",
							ex.Description(),
							updatedCategory.ID(),
							*ex.CategoryID(),
						)
					}
				}
			},
		},
		{
			"modify name",
			"name=Enjoyment",
			false,
			func(t *testing.T, updatedCategory storage.Category, updatedExpenses []storage.Expense) {
				if updatedCategory.Name() != "Enjoyment" {
					t.Fatalf(
						"Category was not updated properly. Expected name to be `Enjoyment` but was %s",
						updatedCategory.Name(),
					)
				}
				for _, ex := range updatedExpenses {
					if ex.Description() == "cinema" {
						if ex.CategoryID() == nil && *ex.CategoryID() != updatedCategory.ID() {
							t.Fatalf(
								"Expense %s was incoreectly updated. Category ID must be %d. Got %d",
								ex.Description(),
								updatedCategory.ID(),
								*ex.CategoryID(),
							)
						}
					}

					if ex.Description() == "gym" {
						if ex.CategoryID() != nil {
							t.Fatalf("Expense was updated. Category ID must be NULL. Got %d", *ex.CategoryID())
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.TestLogger(t)
			s := testutil.SetupTestStorage(t, logger)

			categoryID, err := s.CreateCategory(
				"Entertainment",
				"restaurant|bars|cinema",
			)
			if err != nil {
				t.Fatalf("Failed to create Category: %v", err)
			}

			categories, err := s.GetCategories()
			if err != nil {
				t.Fatalf("Failed to get Categories: %v", err)
			}

			matcher := matcher.New(categories)

			expenses := []storage.Expense{
				storage.NewExpense(
					0,
					"Test Source",
					"cinema",
					"USD",
					-123456,
					time.Now(),
					storage.ChargeType,
					&categoryID,
				),
				storage.NewExpense(0, "Test Source", "gym", "USD", -123, time.Now(), storage.ChargeType, nil),
			}

			_, expenseError := s.InsertExpenses(expenses)
			if expenseError != nil {
				t.Fatalf("Failed to insert test expenses: %v", expenseError)
			}

			handler, router := New(s, matcher, logger)

			oldSyncOnce := router.reportsOnce
			oldMatcher := router.matcher

			body := strings.NewReader(tt.body)
			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/category/%d", categoryID), body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK; got %v", resp.Status)
			}

			ensureNoErrorInTemplateResponse(t, fmt.Sprintf("update category: %s", tt.name), resp.Body)

			categoryUpdated, err := s.GetCategory(categoryID)

			if err != nil {
				t.Fatalf("Failed to get category: %v", err)
			}
			updatedExpenses, err := s.GetExpenses()

			if err != nil {
				t.Fatalf("Failed to get expenses: %v", err)
			}

			tt.assertion(t, categoryUpdated, updatedExpenses)

			if oldSyncOnce == router.reportsOnce {
				t.Error("Router cache was not reset")
			}

			if tt.updateMatcher {
				if oldMatcher == router.matcher {
					t.Error("Router matcher was not updated")
				}
			}
		})
	}
}

func TestUpdateUncategorizedHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory("Entertainment", "restaurant|bars")
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	categories, err := s.GetCategories()
	if err != nil {
		t.Fatalf("Failed to get Categories: %v", err)
	}

	matcher := matcher.New(categories)

	expenses := []storage.Expense{
		storage.NewExpense(
			0,
			"Test Source",
			"cinema. with friends",
			"USD",
			-123456,
			time.Now(),
			storage.ChargeType,
			nil,
		),
	}

	_, expenseError := s.InsertExpenses(expenses)
	if expenseError != nil {
		t.Fatalf("Failed to insert test expenses: %v", expenseError)
	}

	handler, router := New(s, matcher, logger)

	oldSyncOnce := router.reportsOnce

	body := strings.NewReader(fmt.Sprintf("description=cinema. with friends&category_id=%d", categoryID))
	req := httptest.NewRequest(http.MethodPost, "/category/uncategorized/update", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "uncategorized", resp.Body)

	categoryUpdated, err := s.GetCategory(categoryID)
	if err != nil {
		t.Fatalf("Failed to get category: %v", err)
	}

	if categoryUpdated.Pattern() != "restaurant|bars|cinema\\. with friends" {
		t.Fatalf(
			"Category was not updated properly. Expected pattern to be `restaurant|bars|cinema\\. with friends` but was %s",
			categoryUpdated.Pattern(),
		)
	}

	expensesUpdated, err := s.SearchExpensesByDescription("cinema. with friends")

	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(expensesUpdated) != 1 {
		t.Fatalf("Failed to find expenses")
	}

	if expensesUpdated[0].CategoryID() != nil && *expensesUpdated[0].CategoryID() != categoryID {
		t.Fatalf(
			"Expense did not update the category ID. Expected %d but got %d",
			categoryID,
			*expensesUpdated[0].CategoryID(),
		)
	}

	if oldSyncOnce == router.reportsOnce {
		t.Error("Router cache was not reset")
	}
}

func TestResetCategoryHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	cat1ID, err := s.CreateCategory("Food", "restaurant")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	cat2ID, err := s.CreateCategory("Transport", "uber|taxi")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	expenses := []storage.Expense{
		storage.NewExpense(0, "bank", "Restaurant dinner", "EUR", -2500, time.Now(), storage.ChargeType, &cat1ID),
		storage.NewExpense(0, "bank", "Uber ride", "EUR", -1500, time.Now(), storage.ChargeType, &cat2ID),
	}

	_, err = s.InsertExpenses(expenses)
	if err != nil {
		t.Fatalf("Failed to create test expenses: %v", err)
	}

	categories, err := s.GetCategories()
	if err != nil {
		t.Errorf("Failed to get categories: %v", err)
	}
	if len(categories) != 3 {
		t.Fatalf("Expected three categories (two + exclude) initially, got %d", len(categories))
	}

	matcher := matcher.New(categories)
	handler, router := New(s, matcher, logger)

	oldSyncOnce := router.reportsOnce
	oldMatcher := router.matcher

	req := httptest.NewRequest(http.MethodPost, "/category/reset", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", resp.StatusCode, http.StatusOK)
	}

	ensureNoErrorInTemplateResponse(t, "reset categories", resp.Body)

	categories, err = s.GetCategories()
	if err != nil {
		t.Errorf("Failed to get categories after reset: %v", err)
	}
	if len(categories) != 1 {
		t.Errorf("Expected one category (exclude) after reset, got %d", len(categories))
	}

	expenses, getExpensesErr := s.GetExpenses()
	if getExpensesErr != nil {
		t.Errorf("Failed to get expenses after delete: %v", getExpensesErr)
	}
	if len(expenses) != 2 {
		t.Errorf("Expected 2 total expense after delete, got %d", len(expenses))
	}
	if expenses[0].CategoryID() != nil {
		t.Errorf("Expected expense to have null category ID after delete categories")
	}
	if expenses[1].CategoryID() != nil {
		t.Errorf("Expected expense to have null category ID after delete categories")
	}

	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Total Categories") {
		t.Error("Response should contain 'Total Categories' heading")
	}

	if oldSyncOnce == router.reportsOnce {
		t.Error("Router cache was not reset")
	}

	if oldMatcher == router.matcher {
		t.Error("Router matcher was not updated")
	}
}

func TestResetCategoryHandlerEmptyDatabase(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	categories, _ := s.GetCategories()
	matcher := matcher.New(categories)
	handler, router := New(s, matcher, logger)

	oldSyncOnce := router.reportsOnce

	req := httptest.NewRequest(http.MethodPost, "/category/reset", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", resp.StatusCode, http.StatusOK)
	}

	ensureNoErrorInTemplateResponse(t, "reset categories (empty database)", resp.Body)

	categories, err := s.GetCategories()
	if err != nil {
		t.Errorf("Failed to get categories after reset: %v", err)
	}
	if len(categories) != 1 {
		t.Errorf("Expected one category (exclude) after reset, got %d", len(categories))
	}

	if oldSyncOnce == router.reportsOnce {
		t.Error("Router cache was not reset")
	}
}
