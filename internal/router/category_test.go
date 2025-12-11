package router

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestCategoriesHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "categories", resp.Body)
}

func TestCategoryHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	// Create a test category
	categoryID, err := s.CreateCategory(context.Background(), user.ID(), "Entertainment", "cinema|movie", 10000)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/category/%d", categoryID), nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "category edit page", resp.Body)
}

func TestCategoryHandlerNotFound(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	// Request a non-existent category ID
	req := httptest.NewRequest(http.MethodGet, "/category/99999", nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	// Should render with an error message
	body := resp.Body
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyString := string(bodyBytes)

	// The page should render with an error
	if !strings.Contains(bodyString, "error") && !strings.Contains(bodyString, "Error") {
		t.Error("Expected response to contain error message for non-existent category")
	}
}

func TestCategoryHandlerInvalidID(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	// Request with invalid ID format
	req := httptest.NewRequest(http.MethodGet, "/category/invalid", nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	// Should render with an error message
	body := resp.Body
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyString := string(bodyBytes)

	// The page should render with an error
	if !strings.Contains(bodyString, "error") && !strings.Contains(bodyString, "Error") {
		t.Error("Expected response to contain error message for invalid category ID")
	}
}

func TestUncategorizedHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

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

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodGet, "/category/uncategorized", nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
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
	s, user := testutil.SetupTestStorage(t, logger)

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

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler, _ := New(s, logger)

	body := strings.NewReader("q=Coffee")
	req := httptest.NewRequest(http.MethodPost, "/category/uncategorized/search", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
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
	testutil.SetupAuthCookie(t, s, emptyReq, user, sessionCookieName, sessionDuration)
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
	testutil.SetupAuthCookie(t, s, invalidReq, user, sessionCookieName, sessionDuration)
	invalidW := httptest.NewRecorder()

	handler.ServeHTTP(invalidW, invalidReq)

	invalidResp := invalidW.Result()
	if invalidResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK for form parse error; got %v", invalidResp.Status)
	}
}

func TestCreateCategoryHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "cinema", "USD", -123456, time.Now(), storage.ChargeType, nil),
	}

	_, expenseError := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if expenseError != nil {
		t.Fatalf("Failed to insert test expenses: %v", expenseError)
	}

	handler, router := New(s, logger)

	oldMatcher := router.matcher

	body := strings.NewReader("name=Entertainment&pattern=cinema|movie|theater&type=0&monthly_budget=100")
	req := httptest.NewRequest(http.MethodPost, "/category", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "create category", resp.Body)

	categories, err := s.GetCategories(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}

	found := false
	var categoryID int64
	for _, c := range categories {
		if c.Name() == "Entertainment" && c.Pattern() == "cinema|movie|theater" && c.MonthlyBudget() == 10000 {
			found = true
			categoryID = c.ID()
			break
		}
	}

	if !found {
		t.Error("Category was not created")
	}

	expensesUpdated, err := s.SearchExpensesByDescription(context.Background(), user.ID(), "cinema")

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
							"Expense %s was incorrectly updated. Category ID must be %d. Got %d",
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
								"Expense %s was incorrectly updated. Category ID must be %d. Got %d",
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
		{
			"modify budget",
			"monthly_budget=150",
			false,
			func(t *testing.T, updatedCategory storage.Category, _ []storage.Expense) {
				if updatedCategory.MonthlyBudget() != 15000 {
					t.Fatalf(
						"Category was not updated properly. Expected monthly budget to be 150 but was %d",
						updatedCategory.MonthlyBudget(),
					)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.TestLogger(t)
			s, user := testutil.SetupTestStorage(t, logger)

			categoryID, err := s.CreateCategory(context.Background(),
				user.ID(),
				"Entertainment",
				"restaurant|bars|cinema",
				0,
			)
			if err != nil {
				t.Fatalf("Failed to create Category: %v", err)
			}

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

			_, expenseError := s.InsertExpenses(context.Background(), user.ID(), expenses)
			if expenseError != nil {
				t.Fatalf("Failed to insert test expenses: %v", expenseError)
			}

			handler, router := New(s, logger)

			oldMatcher := router.matcher

			body := strings.NewReader(tt.body)
			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/category/%d", categoryID), body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK; got %v", resp.Status)
			}

			ensureNoErrorInTemplateResponse(t, fmt.Sprintf("update category: %s", tt.name), resp.Body)

			categoryUpdated, err := s.GetCategory(context.Background(), user.ID(), categoryID)

			if err != nil {
				t.Fatalf("Failed to get category: %v", err)
			}
			updatedExpenses, err := s.GetExpenses(context.Background(), user.ID())

			if err != nil {
				t.Fatalf("Failed to get expenses: %v", err)
			}

			tt.assertion(t, categoryUpdated, updatedExpenses)

			if tt.updateMatcher {
				if oldMatcher == router.matcher {
					t.Error("Router matcher was not updated")
				}
			}
		})
	}
}

func TestUpdateCategoryPatternDoesNotAffectExcludeCategory(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	entertainmentCategoryID, err := s.CreateCategory(
		context.Background(),
		user.ID(),
		"Entertainment",
		"cinema|movie",
		0,
	)
	if err != nil {
		t.Fatalf("Failed to create Entertainment category: %v", err)
	}

	categories, err := s.GetCategories(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}

	var excludeCategoryID int64
	for _, cat := range categories {
		if cat.Name() == storage.ExcludeCategory {
			excludeCategoryID = cat.ID()
			break
		}
	}
	if excludeCategoryID == 0 {
		t.Fatal("Exclude category not found")
	}

	expenses := []storage.Expense{
		storage.NewExpense(
			0,
			"bank",
			"cinema ticket",
			"USD",
			-1500,
			time.Now(),
			storage.ChargeType,
			&entertainmentCategoryID,
		),
		// Expense that matches the new pattern, but is not updated as it already has a category (exclude)
		storage.NewExpense(
			0,
			"bank",
			"theater hat",
			"USD",
			-5000,
			time.Now(),
			storage.ChargeType,
			&excludeCategoryID,
		),
		// Internal transfer excluded
		storage.NewExpense(
			0,
			"bank",
			"internal transfer",
			"USD",
			-5000,
			time.Now(),
			storage.ChargeType,
			&excludeCategoryID,
		),
		// Income in exclude category
		storage.NewExpense(0, "bank", "salary refund", "USD", 3000, time.Now(), storage.IncomeType, &excludeCategoryID),
		// Income with no category that should match the pattern, but won't get updated
		storage.NewExpense(0, "bank", "cinema refund", "USD", 3000, time.Now(), storage.IncomeType, nil),
		// Uncategorized expense that should match new pattern
		storage.NewExpense(0, "bank", "theater show", "USD", -2000, time.Now(), storage.ChargeType, nil),
		// Uncategorized expense that should not match
		storage.NewExpense(0, "bank", "grocery shopping", "USD", -4000, time.Now(), storage.ChargeType, nil),
	}

	_, err = s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler, _ := New(s, logger)

	// Update the entertainment category pattern to include "theater"
	body := strings.NewReader("pattern=cinema|movie|theater")
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/category/%d", entertainmentCategoryID), body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	// Verify the response doesn't contain errors
	ensureNoErrorInTemplateResponse(t, "update category pattern", resp.Body)

	// Check that all expenses are still correctly categorized
	allExpenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses after update: %v", err)
	}

	// Track what we find
	var cinemaExpense, transferExpense, salaryIncome, cinemaRefund, theaterExpense, excludeTheaterExpense, groceryExpense storage.Expense
	for i, exp := range allExpenses {
		switch exp.Description() {
		case "cinema ticket":
			cinemaExpense = allExpenses[i]
		case "internal transfer":
			transferExpense = allExpenses[i]
		case "salary refund":
			salaryIncome = allExpenses[i]
		case "cinema refund":
			cinemaRefund = allExpenses[i]
		case "theater show":
			theaterExpense = allExpenses[i]
		case "theater hat":
			excludeTheaterExpense = allExpenses[i]
		case "grocery shopping":
			groceryExpense = allExpenses[i]
		}
	}

	// Verify cinema expense is still in entertainment category
	if cinemaExpense == nil {
		t.Fatal("Cinema expense not found")
	}
	if cinemaExpense.CategoryID() == nil || *cinemaExpense.CategoryID() != entertainmentCategoryID {
		t.Errorf("Cinema expense should be in entertainment category, got: %v", cinemaExpense.CategoryID())
	}

	// Verify theater expense was moved to entertainment category
	if theaterExpense == nil {
		t.Fatal("Theater expense not found")
	}
	if theaterExpense.CategoryID() == nil || *theaterExpense.CategoryID() != entertainmentCategoryID {
		t.Errorf("Theater expense should be in entertainment category, got: %v", theaterExpense.CategoryID())
	}

	// Verify excluded theater expense remains excluded
	if excludeTheaterExpense == nil {
		t.Fatal("Theater expense not found")
	}
	if excludeTheaterExpense.CategoryID() == nil || *excludeTheaterExpense.CategoryID() != excludeCategoryID {
		t.Errorf("Excluded theater expense should be in excluded category, got: %v", excludeTheaterExpense.CategoryID())
	}

	// Verify exclude category expenses are NOT affected
	if transferExpense == nil {
		t.Fatal("Transfer expense not found")
	}
	if transferExpense.CategoryID() == nil || *transferExpense.CategoryID() != excludeCategoryID {
		t.Errorf("Transfer expense should remain in exclude category, got: %v", transferExpense.CategoryID())
	}

	// Verify uncategorized income is NOT affected
	if cinemaRefund == nil {
		t.Fatal("cinema income not found")
	}
	if cinemaRefund.CategoryID() != nil {
		t.Errorf("Cinema income should remain uncategorized, got: %v", cinemaRefund.CategoryID())
	}

	// Verify exclude category income is NOT affected
	if salaryIncome == nil {
		t.Fatal("Salary income not found")
	}
	if salaryIncome.CategoryID() == nil || *salaryIncome.CategoryID() != excludeCategoryID {
		t.Errorf("Salary income should remain in exclude category, got: %v", salaryIncome.CategoryID())
	}

	// Verify grocery expense remains uncategorized
	if groceryExpense == nil {
		t.Fatal("Grocery expense not found")
	}
	if groceryExpense.CategoryID() != nil {
		t.Errorf("Grocery expense should remain uncategorized, got: %v", *groceryExpense.CategoryID())
	}
}

func TestUpdateUncategorizedHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory(context.Background(), user.ID(), "Entertainment", "restaurant|bars", 0)
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

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

	_, expenseError := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if expenseError != nil {
		t.Fatalf("Failed to insert test expenses: %v", expenseError)
	}

	handler, _ := New(s, logger)

	body := strings.NewReader(fmt.Sprintf("description=cinema. with friends&category_id=%d", categoryID))
	req := httptest.NewRequest(http.MethodPost, "/category/uncategorized/update", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "uncategorized", resp.Body)

	categoryUpdated, err := s.GetCategory(context.Background(), user.ID(), categoryID)
	if err != nil {
		t.Fatalf("Failed to get category: %v", err)
	}

	if categoryUpdated.Pattern() != "restaurant|bars|cinema\\. with friends" {
		t.Fatalf(
			"Category was not updated properly. Expected pattern to be `restaurant|bars|cinema\\. with friends` but was %s",
			categoryUpdated.Pattern(),
		)
	}

	expensesUpdated, err := s.SearchExpensesByDescription(context.Background(), user.ID(), "cinema. with friends")

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
}

func TestResetCategoryHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	cat1ID, err := s.CreateCategory(context.Background(), user.ID(), "Food", "restaurant", 0)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	cat2ID, err := s.CreateCategory(context.Background(), user.ID(), "Transport", "uber|taxi", 0)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	expenses := []storage.Expense{
		storage.NewExpense(0, "bank", "Restaurant dinner", "EUR", -2500, time.Now(), storage.ChargeType, &cat1ID),
		storage.NewExpense(0, "bank", "Uber ride", "EUR", -1500, time.Now(), storage.ChargeType, &cat2ID),
	}

	_, err = s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to create test expenses: %v", err)
	}

	categories, err := s.GetCategories(context.Background(), user.ID())
	if err != nil {
		t.Errorf("Failed to get categories: %v", err)
	}
	if len(categories) != 3 {
		t.Fatalf("Expected three categories (two + exclude) initially, got %d", len(categories))
	}

	handler, router := New(s, logger)

	oldMatcher := router.matcher

	req := httptest.NewRequest(http.MethodPost, "/category/reset", nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", resp.StatusCode, http.StatusOK)
	}

	ensureNoErrorInTemplateResponse(t, "reset categories", resp.Body)

	categories, err = s.GetCategories(context.Background(), user.ID())
	if err != nil {
		t.Errorf("Failed to get categories after reset: %v", err)
	}
	if len(categories) != 1 {
		t.Errorf("Expected one category (exclude) after reset, got %d", len(categories))
	}

	expenses, getExpensesErr := s.GetExpenses(context.Background(), user.ID())
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

	if oldMatcher == router.matcher {
		t.Error("Router matcher was not updated")
	}
}

func TestResetCategoryHandlerEmptyDatabase(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	req := httptest.NewRequest(http.MethodPost, "/category/reset", nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", resp.StatusCode, http.StatusOK)
	}

	ensureNoErrorInTemplateResponse(t, "reset categories (empty database)", resp.Body)

	categories, err := s.GetCategories(context.Background(), user.ID())
	if err != nil {
		t.Errorf("Failed to get categories after reset: %v", err)
	}
	if len(categories) != 1 {
		t.Errorf("Expected one category (exclude) after reset, got %d", len(categories))
	}
}

func TestParseCategoryForm(t *testing.T) {
	tests := []struct {
		name          string
		formData      string
		expectError   bool
		expectedData  *categoryFormData
		errorContains string
	}{
		{
			name:        "valid form with budget",
			formData:    "name=Entertainment&pattern=cinema|movie&monthly_budget=100.50",
			expectError: false,
			expectedData: &categoryFormData{
				Name:          "Entertainment",
				Pattern:       "cinema|movie",
				MonthlyBudget: 10050, // cents
			},
		},
		{
			name:        "valid form without budget",
			formData:    "name=Food&pattern=restaurant|cafe",
			expectError: false,
			expectedData: &categoryFormData{
				Name:          "Food",
				Pattern:       "restaurant|cafe",
				MonthlyBudget: 0,
			},
		},
		{
			name:        "valid form with empty budget",
			formData:    "name=Transport&pattern=uber|taxi&monthly_budget=",
			expectError: false,
			expectedData: &categoryFormData{
				Name:          "Transport",
				Pattern:       "uber|taxi",
				MonthlyBudget: 0,
			},
		},
		{
			name:        "valid form with zero budget",
			formData:    "name=Misc&pattern=misc&monthly_budget=0",
			expectError: false,
			expectedData: &categoryFormData{
				Name:          "Misc",
				Pattern:       "misc",
				MonthlyBudget: 0,
			},
		},
		{
			name:          "missing name",
			formData:      "pattern=test&monthly_budget=100",
			expectError:   true,
			errorContains: "name and a valid regex pattern",
		},
		{
			name:          "missing pattern",
			formData:      "name=Test&monthly_budget=100",
			expectError:   true,
			errorContains: "name and a valid regex pattern",
		},
		{
			name:          "empty name",
			formData:      "name=&pattern=test",
			expectError:   true,
			errorContains: "name and a valid regex pattern",
		},
		{
			name:          "empty pattern",
			formData:      "name=Test&pattern=",
			expectError:   true,
			errorContains: "name and a valid regex pattern",
		},
		{
			name:          "invalid regex pattern - unclosed bracket",
			formData:      "name=Test&pattern=[invalid",
			expectError:   true,
			errorContains: "invalid pattern",
		},
		{
			name:          "invalid regex pattern - unclosed paren",
			formData:      "name=Test&pattern=(invalid",
			expectError:   true,
			errorContains: "invalid pattern",
		},
		{
			name:          "invalid budget format",
			formData:      "name=Test&pattern=valid&monthly_budget=abc",
			expectError:   true,
			errorContains: "invalid budget format",
		},
		{
			name:          "negative budget",
			formData:      "name=Test&pattern=valid&monthly_budget=-100",
			expectError:   true,
			errorContains: "budget cannot be negative",
		},
		{
			name:        "budget with decimal precision",
			formData:    "name=Test&pattern=valid&monthly_budget=99.99",
			expectError: false,
			expectedData: &categoryFormData{
				Name:          "Test",
				Pattern:       "valid",
				MonthlyBudget: 9999, // cents
			},
		},
		{
			name:        "complex regex pattern",
			formData:    "name=Shopping&pattern=^(amazon|ebay|shop)",
			expectError: false,
			expectedData: &categoryFormData{
				Name:          "Shopping",
				Pattern:       "^(amazon|ebay|shop)",
				MonthlyBudget: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.NewReader(tt.formData)
			req := httptest.NewRequest(http.MethodPost, "/category", body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			result, err := parseCategoryForm(req)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result == nil {
				t.Errorf("Expected result but got nil")
				return
			}
			if result.Name != tt.expectedData.Name {
				t.Errorf("Expected name %q, got %q", tt.expectedData.Name, result.Name)
			}
			if result.Pattern != tt.expectedData.Pattern {
				t.Errorf("Expected pattern %q, got %q", tt.expectedData.Pattern, result.Pattern)
			}
			if result.MonthlyBudget != tt.expectedData.MonthlyBudget {
				t.Errorf("Expected budget %d, got %d", tt.expectedData.MonthlyBudget, result.MonthlyBudget)
			}
		})
	}
}
