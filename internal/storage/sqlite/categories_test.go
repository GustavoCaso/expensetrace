package sqlite

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func TestCreateCategoriesTable(t *testing.T) {
	stor := setupTestStorage(t)

	// Verify we can create a category (which means table exists)
	_, err := stor.CreateCategory("Test", "test.*")
	if err != nil {
		t.Errorf("Failed to create test category: %v", err)
	}
}
func TestGetCategories(t *testing.T) {
	stor := setupTestStorage(t)

	// Create test categories
	testCategories := []struct {
		name    string
		pattern string
	}{
		{"Food", "restaurant|food"},
		{"Transport", "uber|taxi"},
		{"Entertainment", "netflix|spotify"},
	}

	for _, cat := range testCategories {
		_, err := stor.CreateCategory(cat.name, cat.pattern)
		if err != nil {
			t.Fatalf("Failed to create test category: %v", err)
		}
	}

	categories, err := stor.GetCategories()
	if err != nil {
		t.Errorf("Failed to get categories: %v", err)
	}

	if len(categories) != len(testCategories) {
		t.Errorf("Expected %d categories, got %d", len(testCategories), len(categories))
	}

	for i, cat := range categories {
		if cat.Name() != testCategories[i].name {
			t.Errorf("Category[%d].Name = %v, want %v", i, cat.Name(), testCategories[i].name)
		}
		if cat.Pattern() != testCategories[i].pattern {
			t.Errorf("Category[%d].Pattern = %v, want %v", i, cat.Pattern(), testCategories[i].pattern)
		}
	}
}

func TestGetCategory(t *testing.T) {
	stor := setupTestStorage(t)

	// Create test category
	id, err := stor.CreateCategory("Test", "test.*")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	// Test getting existing category
	category, err := stor.GetCategory(id)
	if err != nil {
		t.Errorf("Failed to get category: %v", err)
	}

	if category.Name() != "Test" {
		t.Errorf("Category.Name = %v, want Test", category.Name())
	}
	if category.Pattern() != "test.*" {
		t.Errorf("Category.Pattern = %v, want test.*", category.Pattern())
	}

	// Test getting non-existent category
	_, err = stor.GetCategory(999)
	if err == nil {
		t.Error("Expected error when getting non-existent category, got nil")
	}

	if !errors.Is(err, &storage.NotFoundError{}) {
		t.Error("Expected error to be of type storage.NotFoundError")
	}
}

func TestCreateCategory(t *testing.T) {
	stor := setupTestStorage(t)

	id, err := stor.CreateCategory("Test", "test.*")
	if err != nil {
		t.Errorf("Failed to create category: %v", err)
	}

	if id != 1 {
		t.Errorf("Expected category ID 1, got %d", id)
	}

	// Verify category was created
	category, err := stor.GetCategory(id)
	if err != nil {
		t.Errorf("Failed to get created category: %v", err)
	}

	if category.Name() != "Test" {
		t.Errorf("Created category.Name = %v, want Test", category.Name())
	}
	if category.Pattern() != "test.*" {
		t.Errorf("Created category.Pattern = %v, want test.*", category.Pattern())
	}
}

func TestDeleteCategories(t *testing.T) {
	stor := setupTestStorage(t)

	// Create test categories
	_, err := stor.CreateCategory("Food", "restaurant|food")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	_, err = stor.CreateCategory("Transport", "uber|taxi")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	// Drop all categories
	rowsAffected, err := stor.DeleteCategories()
	if err != nil {
		t.Errorf("Failed to drop categories: %v", err)
	}

	if rowsAffected != 2 {
		t.Errorf("Expected 2 rows affected, got %d", rowsAffected)
	}

	// Verify categories are deleted
	categories, err := stor.GetCategories()
	if err != nil {
		t.Errorf("Failed to get categories after drop: %v", err)
	}
	if len(categories) != 0 {
		t.Errorf("Expected 0 categories after drop, got %d", len(categories))
	}
}

func TestDeleteCategoriesWithExpenses(t *testing.T) {
	stor := setupTestStorage(t)

	catID, createCategoryErr := stor.CreateCategory("Food", "restaurant")
	if createCategoryErr != nil {
		t.Fatalf("Failed to create test category: %v", createCategoryErr)
	}

	testTime := time.Now()
	expense := storage.NewExpense(0, "bank", "Restaurant dinner", "EUR", -2500, testTime, storage.ChargeType, &catID)

	expenses := []storage.Expense{expense}
	_, insertErr := stor.InsertExpenses(expenses)
	if insertErr != nil {
		t.Fatalf("Failed to create test expense: %v", insertErr)
	}

	// Verify the expense has a category before deletion
	allExpenses, err := stor.GetExpenses()
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}
	if len(allExpenses) != 1 {
		t.Fatalf("Expected 1 expense, got %d", len(allExpenses))
	}
	if allExpenses[0].CategoryID() == nil {
		t.Fatalf("Expected expense to have a category before deletion")
	}

	rowsAffected, deleteCategoryErr := stor.DeleteCategories()
	if deleteCategoryErr != nil {
		t.Errorf("Failed to drop categories: %v", deleteCategoryErr)
	}

	if rowsAffected != 1 {
		t.Errorf("Expected 1 category deleted, got %d", rowsAffected)
	}

	categories, getCategoriesErr := stor.GetCategories()

	if getCategoriesErr != nil {
		t.Errorf("failed to get categories after deleting them %s", getCategoriesErr.Error())
	}

	if len(categories) != 0 {
		t.Errorf("got categories after deleting them %d", len(categories))
	}

	expensesAfter, getExpensesErr := stor.GetExpenses()
	if getExpensesErr != nil {
		t.Errorf("Failed to get expenses after drop: %v", getExpensesErr)
	}
	if len(expensesAfter) != 1 {
		t.Errorf("Expected 1 total expense after drop, got %d", len(expensesAfter))
	}
	if expensesAfter[0].CategoryID() != nil {
		t.Errorf(
			"Expected expense to have zero category ID after delete categories, got %+v",
			expensesAfter[0].CategoryID(),
		)
	}
}

func TestUpdateCategory(t *testing.T) {
	stor := setupTestStorage(t)

	catID, err := stor.CreateCategory("Food", "restaurant")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	err = stor.UpdateCategory(catID, "Dining", "restaurant|dining|food")
	if err != nil {
		t.Errorf("Failed to update category: %v", err)
	}

	category, err := stor.GetCategory(catID)
	if err != nil {
		t.Errorf("Failed to get updated category: %v", err)
	}

	if category.Name() != "Dining" {
		t.Errorf("Expected category name 'Dining', got '%s'", category.Name())
	}
	if category.Pattern() != "restaurant|dining|food" {
		t.Errorf("Expected pattern 'restaurant|dining|food', got '%s'", category.Pattern())
	}
}

func setupTestStorage(t *testing.T) storage.Storage {
	t.Helper()
	// We use a tempDir + the unique test name (t.Name) that way we can warrant that any test has its own DB
	// Using a tempDir ensure it gets clean after each test
	sqlFile := filepath.Join(t.TempDir(), fmt.Sprintf(":memory:%s", strings.ReplaceAll(t.Name(), "/", ":")))
	stor, err := New(sqlFile)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}

	logger := logger.New(logger.Config{})
	err = stor.ApplyMigrations(logger)
	if err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	t.Cleanup(func() {
		if err = stor.Close(); err != nil {
			t.Errorf("Failed to close test storage: %v", err)
		}
	})

	return stor
}
