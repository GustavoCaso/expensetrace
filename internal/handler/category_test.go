package handler

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

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
	}
	matcher := category.NewMatcher(categories)

	handler := New(database, matcher, logger)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	w := httptest.NewRecorder()

	// Serve request
	handler.HTTPHandler.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}
}

func TestUncategorizedHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
	}
	matcher := category.NewMatcher(categories)

	// Create test expenses
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

	// Create server
	handler := New(database, matcher, logger)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/uncategorized", nil)
	w := httptest.NewRecorder()

	// Serve request
	handler.HTTPHandler.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}
}

func TestCreateCategoryHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
	}
	matcher := category.NewMatcher(categories)

	// Create test expenses
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

	handler := New(database, matcher, logger)

	// Create test request
	body := strings.NewReader("name=Entertainment&pattern=cinema|movie|theater&type=0")
	req := httptest.NewRequest(http.MethodPost, "/category", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Serve request
	handler.HTTPHandler.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	// Verify category was created
	categories, err := db.GetCategories(database)
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}

	found := false
	var categoryID int
	for _, c := range categories {
		if c.Name == "Entertainment" && c.Pattern == "cinema|movie|theater" {
			if c.Type != db.ExpenseCategoryType {
				t.Fatalf("Category type was not set properly. Expected %d but got %d", db.ExpenseCategoryType, c.Type)
			}
			found = true
			categoryID = c.ID
			break
		}
	}

	if !found {
		t.Error("Category was not created")
	}

	// Verify expense was updated
	expensesUpdated, err := db.SearchExpensesByDescription(database, "cinema")

	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(expensesUpdated) > 1 {
		t.Fatalf("Failed more expenses than it should: %v", err)
	}

	if expensesUpdated[0].CategoryID.Int64 != int64(categoryID) {
		t.Fatal("Expense did not update the category ID")
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
					if int(ex.CategoryID.Int64) != updatedCategory.ID {
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
						if int(ex.CategoryID.Int64) != updatedCategory.ID {
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
			// Create test categories
			categoryID, err := db.CreateCategory(
				database,
				"Entertainment",
				"restaurant|bars|cinema",
				db.ExpenseCategoryType,
			)
			if err != nil {
				t.Fatalf("Failed to create Category: %v", err)
			}

			categories, err := db.GetCategories(database)
			if err != nil {
				t.Fatalf("Failed to get Categories: %v", err)
			}

			matcher := category.NewMatcher(categories)

			// Create test expenses
			expenses := []*db.Expense{
				// id 1
				{
					Source:      "Test Source",
					Date:        time.Now(),
					Description: "cinema",
					Amount:      -123456,
					Type:        db.ChargeType,
					Currency:    "USD",
					CategoryID:  sql.NullInt64{Int64: categoryID, Valid: true},
				},
				// id 2
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

			handler := New(database, matcher, logger)

			// Create test request
			body := strings.NewReader(tt.body)
			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/category/%d", categoryID), body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			// Serve request
			handler.HTTPHandler.ServeHTTP(w, req)

			// Check response
			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK; got %v", resp.Status)
			}

			// Get updated category
			categoryUpdated, err := db.GetCategory(database, categoryID)

			if err != nil {
				t.Fatalf("Failed to get category: %v", err)
			}
			updatedExpenses, err := db.GetExpenses(database)

			if err != nil {
				t.Fatalf("Failed to get expenses: %v", err)
			}

			tt.assertion(t, categoryUpdated, updatedExpenses)
		})
	}
}

func TestUpdateUncategorizedHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	// Create test categories
	categoryID, err := db.CreateCategory(database, "Entertainment", "restaurant|bars", db.ExpenseCategoryType)
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	categories, err := db.GetCategories(database)
	if err != nil {
		t.Fatalf("Failed to get Categories: %v", err)
	}

	matcher := category.NewMatcher(categories)

	// Create test expenses
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

	handler := New(database, matcher, logger)

	// Create test request
	body := strings.NewReader(fmt.Sprintf("description=cinema. with friends&categoryID=%d", categoryID))
	req := httptest.NewRequest(http.MethodPost, "/category/uncategorized/update", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Serve request
	handler.HTTPHandler.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	// Verify category was updated
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

	// Verify expense was updated
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
}
