package router

import (
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
	database := testutil.SetupTestDB(t)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
	}
	matcher := category.NewMatcher(categories)

	// Create router
	handler, _ := New(database, matcher)

	// Create test request
	req := httptest.NewRequest("GET", "/categories", nil)
	w := httptest.NewRecorder()

	// Serve request
	handler.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}
}

func TestUncategorizedHandler(t *testing.T) {
	database := testutil.SetupTestDB(t)

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

	err := db.InsertExpenses(database, expenses)
	if len(err) > 0 {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	// Create router
	handler, _ := New(database, matcher)

	// Create test request
	req := httptest.NewRequest("GET", "/uncategorized", nil)
	w := httptest.NewRecorder()

	// Serve request
	handler.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}
}

func TestCreateCategoryHandler(t *testing.T) {
	database := testutil.SetupTestDB(t)

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

	expenseErrors := db.InsertExpenses(database, expenses)
	if len(expenseErrors) > 0 {
		t.Fatalf("Failed to insert test expenses: %v", expenseErrors)
	}

	// Create router
	handler, router := New(database, matcher)

	oldMatcher := router.matcher
	oldSyncOnce := router.reportsOnce

	// Create test request
	body := strings.NewReader("name=Entertainment&pattern=cinema|movie|theater")
	req := httptest.NewRequest("POST", "/category", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Serve request
	handler.ServeHTTP(w, req)

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
	var categoryId int
	for _, c := range categories {
		if c.Name == "Entertainment" && c.Pattern == "cinema|movie|theater" {
			found = true
			categoryId = c.ID
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

	if expensesUpdated[0].CategoryID != categoryId {
		t.Fatal("Expense did not update the category ID")
	}

	if oldMatcher == router.matcher {
		t.Error("Category matcher was not re-created")
	}

	if oldSyncOnce == router.reportsOnce {
		t.Error("Router cache was not reset")
	}
}

func TestUpdateCategoryHandler(t *testing.T) {
	database := testutil.SetupTestDB(t)

	// Create test categories
	categoryID, err := db.CreateCategory(database, "Entertainment", "restaurant|bars")
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
			Description: "cinema",
			Amount:      -123456,
			Type:        db.ChargeType,
			Currency:    "USD",
		},
	}

	expenseErrors := db.InsertExpenses(database, expenses)
	if len(expenseErrors) > 0 {
		t.Fatalf("Failed to insert test expenses: %v", expenseErrors)
	}

	// Create router
	handler, router := New(database, matcher)

	oldSyncOnce := router.reportsOnce

	// Create test request
	body := strings.NewReader(fmt.Sprintf("description=cinema&categoryID=%d", categoryID))
	req := httptest.NewRequest("POST", "/category/uncategorized/update", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Serve request
	handler.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	// Verify expense was updated
	expensesUpdated, err := db.SearchExpensesByDescription(database, "cinema")

	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(expensesUpdated) > 1 {
		t.Fatalf("Failed more expenses than it should: %v", err)
	}

	if expensesUpdated[0].CategoryID != int(categoryID) {
		t.Fatal("Expense did not update the category ID")
	}

	if oldSyncOnce == router.reportsOnce {
		t.Error("Router cache was not reset")
	}
}
