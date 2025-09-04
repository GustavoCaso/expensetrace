package router

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestCategoriesHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
	}
	matcher := category.NewMatcher(categories)

	handler, _ := New(database, matcher, logger)

	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}
}

func TestUncategorizedHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
	}
	matcher := category.NewMatcher(categories)

	expenses := []*db.Expense{
		{
			Source:      "Test Source",
			Date:        time.Now(),
			Description: "Uncategorized expense",
			Amount:      -123456,
			Type:        db.ChargeType,
			Currency:    "USD",
		},
	}

	_, err := db.InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler, _ := New(database, matcher, logger)

	req := httptest.NewRequest(http.MethodGet, "/uncategorized", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}
}

func TestCreateCategoryHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
	}
	matcher := category.NewMatcher(categories)

	expenses := []*db.Expense{
		{
			Source:      "Test Source",
			Date:        time.Now(),
			Description: "cinema",
			Amount:      -123456,
			Type:        db.ChargeType,
			Currency:    "USD",
		},
	}

	_, expenseError := db.InsertExpenses(database, expenses)
	if expenseError != nil {
		t.Fatalf("Failed to insert test expenses: %v", expenseError)
	}

	handler, router := New(database, matcher, logger)

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

	categories, err := db.GetCategories(database)
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}

	found := false
	var categoryID int64
	for _, c := range categories {
		if c.Name == "Entertainment" && c.Pattern == "cinema|movie|theater" {
			found = true
			categoryID = c.ID
			break
		}
	}

	if !found {
		t.Error("Category was not created")
	}

	expensesUpdated, err := db.SearchExpensesByDescription(database, "cinema")

	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(expensesUpdated) > 1 {
		t.Fatalf("Failed more expenses than it should: %v", err)
	}

	if expensesUpdated[0].CategoryID.Int64 != categoryID {
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
		assertion     func(t *testing.T, updatedCategory db.Category, updatedExpenses []*db.Expense)
	}{
		{
			"modify pattern and set expense to NULL category",
			"pattern=test_pattern",
			true,
			func(t *testing.T, updatedCategory db.Category, updatedExpenses []*db.Expense) {
				if updatedCategory.Pattern != "test_pattern" {
					t.Fatalf(
						"Category was not updated properly. Expected pattern to be `test_pattern` but was %s",
						updatedCategory.Pattern,
					)
				}

				for _, ex := range updatedExpenses {
					if ex.Description == "cinema" {
						if ex.CategoryID.Valid {
							t.Fatalf(
								"Expense was not properly updated. Category ID must be NULL. Got %d",
								ex.CategoryID.Int64,
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
			func(t *testing.T, updatedCategory db.Category, updatedExpenses []*db.Expense) {
				if updatedCategory.Pattern != "restaurant|bars|cinema|gym" {
					t.Fatalf(
						"Category was not updated properly. Expected pattern to be `restaurant|bars|cinema|gym` but was %s",
						updatedCategory.Pattern,
					)
				}

				for _, ex := range updatedExpenses {
					if ex.CategoryID.Int64 != updatedCategory.ID {
						t.Fatalf(
							"Expense %s was incoreectly updated. Category ID must be %d. Got %d",
							ex.Description,
							updatedCategory.ID,
							ex.CategoryID.Int64,
						)
					}
				}
			},
		},
		{
			"modify name",
			"name=Enjoyment",
			false,
			func(t *testing.T, updatedCategory db.Category, updatedExpenses []*db.Expense) {
				if updatedCategory.Name != "Enjoyment" {
					t.Fatalf(
						"Category was not updated properly. Expected name to be `Enjoyment` but was %s",
						updatedCategory.Name,
					)
				}
				for _, ex := range updatedExpenses {
					if ex.Description == "cinema" {
						if ex.CategoryID.Int64 != updatedCategory.ID {
							t.Fatalf(
								"Expense %s was incoreectly updated. Category ID must be %d. Got %d",
								ex.Description,
								updatedCategory.ID,
								ex.CategoryID.Int64,
							)
						}
					}

					if ex.Description == "gym" {
						if ex.CategoryID.Valid {
							t.Fatalf("Expense was updated. Category ID must be NULL. Got %d", ex.CategoryID.Int64)
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.TestLogger(t)
			database := testutil.SetupTestDB(t, logger)

			categoryID, err := db.CreateCategory(
				database,
				"Entertainment",
				"restaurant|bars|cinema",
			)
			if err != nil {
				t.Fatalf("Failed to create Category: %v", err)
			}

			categories, err := db.GetCategories(database)
			if err != nil {
				t.Fatalf("Failed to get Categories: %v", err)
			}

			matcher := category.NewMatcher(categories)

			expenses := []*db.Expense{

				{
					Source:      "Test Source",
					Date:        time.Now(),
					Description: "cinema",
					Amount:      -123456,
					Type:        db.ChargeType,
					Currency:    "USD",
					CategoryID:  sql.NullInt64{Int64: categoryID, Valid: true},
				},

				{
					Source:      "Test Source",
					Date:        time.Now(),
					Description: "gym",
					Amount:      -123,
					Type:        db.ChargeType,
					Currency:    "USD",
					CategoryID:  sql.NullInt64{Int64: int64(0), Valid: false},
				},
			}

			_, expenseError := db.InsertExpenses(database, expenses)
			if expenseError != nil {
				t.Fatalf("Failed to insert test expenses: %v", expenseError)
			}

			handler, router := New(database, matcher, logger)

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

			categoryUpdated, err := db.GetCategory(database, categoryID)

			if err != nil {
				t.Fatalf("Failed to get category: %v", err)
			}
			updatedExpenses, err := db.GetExpenses(database)

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
	database := testutil.SetupTestDB(t, logger)

	categoryID, err := db.CreateCategory(database, "Entertainment", "restaurant|bars")
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	categories, err := db.GetCategories(database)
	if err != nil {
		t.Fatalf("Failed to get Categories: %v", err)
	}

	matcher := category.NewMatcher(categories)

	expenses := []*db.Expense{
		{
			Source:      "Test Source",
			Date:        time.Now(),
			Description: "cinema. with friends",
			Amount:      -123456,
			Type:        db.ChargeType,
			Currency:    "USD",
		},
	}

	_, expenseError := db.InsertExpenses(database, expenses)
	if expenseError != nil {
		t.Fatalf("Failed to insert test expenses: %v", expenseError)
	}

	handler, router := New(database, matcher, logger)

	oldSyncOnce := router.reportsOnce

	body := strings.NewReader(fmt.Sprintf("description=cinema. with friends&categoryID=%d", categoryID))
	req := httptest.NewRequest(http.MethodPost, "/category/uncategorized/update", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	categoryUpdated, err := db.GetCategory(database, categoryID)
	if err != nil {
		t.Fatalf("Failed to get category: %v", err)
	}

	if categoryUpdated.Pattern != "restaurant|bars|cinema\\. with friends" {
		t.Fatalf(
			"Category was not updated properly. Expected pattern to be `restaurant|bars|cinema\\. with friends` but was %s",
			categoryUpdated.Pattern,
		)
	}

	expensesUpdated, err := db.SearchExpensesByDescription(database, "cinema. with friends")

	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(expensesUpdated) != 1 {
		t.Fatalf("Failed to find expenses")
	}

	if int(expensesUpdated[0].CategoryID.Int64) != int(categoryID) {
		t.Fatalf(
			"Expense did not update the category ID. Expected %d but got %d",
			categoryID,
			expensesUpdated[0].CategoryID.Int64,
		)
	}

	if oldSyncOnce == router.reportsOnce {
		t.Error("Router cache was not reset")
	}
}

func TestResetCategoryHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	cat1ID, err := db.CreateCategory(database, "Food", "restaurant")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	cat2ID, err := db.CreateCategory(database, "Transport", "uber|taxi")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	expenses := []*db.Expense{
		{
			Source:      "bank",
			Date:        time.Now(),
			Description: "Restaurant dinner",
			Amount:      -2500,
			Type:        db.ChargeType,
			Currency:    "EUR",
			CategoryID:  sql.NullInt64{Int64: cat1ID, Valid: true},
		},
		{
			Source:      "bank",
			Date:        time.Now(),
			Description: "Uber ride",
			Amount:      -1500,
			Type:        db.ChargeType,
			Currency:    "EUR",
			CategoryID:  sql.NullInt64{Int64: cat2ID, Valid: true},
		},
	}

	_, err = db.InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to create test expenses: %v", err)
	}

	categories, err := db.GetCategories(database)
	if err != nil {
		t.Errorf("Failed to get categories: %v", err)
	}
	if len(categories) != 2 {
		t.Fatalf("Expected 2 categories initially, got %d", len(categories))
	}

	matcher := category.NewMatcher(categories)
	handler, router := New(database, matcher, logger)

	oldSyncOnce := router.reportsOnce
	oldMatcher := router.matcher

	req := httptest.NewRequest(http.MethodPost, "/category/reset", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", resp.StatusCode, http.StatusOK)
	}

	categories, err = db.GetCategories(database)
	if err != nil {
		t.Errorf("Failed to get categories after reset: %v", err)
	}
	if len(categories) != 0 {
		t.Errorf("Expected 0 categories after reset, got %d", len(categories))
	}

	expenses, getExpensesErr := db.GetExpenses(database)
	if getExpensesErr != nil {
		t.Errorf("Failed to get expenses after delete: %v", getExpensesErr)
	}
	if len(expenses) != 2 {
		t.Errorf("Expected 2 total expense after delete, got %d", len(expenses))
	}
	if expenses[0].CategoryID.Valid {
		t.Errorf("Expected expense to have null category ID after delete categories")
	}
	if expenses[1].CategoryID.Valid {
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
	database := testutil.SetupTestDB(t, logger)

	categories, _ := db.GetCategories(database)
	matcher := category.NewMatcher(categories)
	handler, router := New(database, matcher, logger)

	oldSyncOnce := router.reportsOnce

	req := httptest.NewRequest(http.MethodPost, "/category/reset", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", resp.StatusCode, http.StatusOK)
	}

	categories, err := db.GetCategories(database)
	if err != nil {
		t.Errorf("Failed to get categories after reset: %v", err)
	}
	if len(categories) != 0 {
		t.Errorf("Expected 0 categories after reset, got %d", len(categories))
	}

	if oldSyncOnce == router.reportsOnce {
		t.Error("Router cache was not reset")
	}
}
