package db

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateCategoriesTable(t *testing.T) {
	db := setupTestDB(t)

	// Verify table exists by trying to insert a record
	_, err := db.Exec("INSERT INTO categories(name, pattern) VALUES(?, ?)", "Test", "test.*")
	if err != nil {
		t.Errorf("Failed to insert test category: %v", err)
	}
}
func TestGetCategories(t *testing.T) {
	db := setupTestDB(t)

	// Insert test categories
	testCategories := []struct {
		name    string
		pattern string
	}{
		{"Food", "restaurant|food"},
		{"Transport", "uber|taxi"},
		{"Entertainment", "netflix|spotify"},
	}

	for _, cat := range testCategories {
		_, err := db.Exec("INSERT INTO categories(name, pattern) VALUES(?, ?)", cat.name, cat.pattern)
		if err != nil {
			t.Fatalf("Failed to insert test category: %v", err)
		}
	}

	categories, err := GetCategories(db)
	if err != nil {
		t.Errorf("Failed to get categories: %v", err)
	}

	if len(categories) != len(testCategories) {
		t.Errorf("Expected %d categories, got %d", len(testCategories), len(categories))
	}

	for i, cat := range categories {
		if cat.Name != testCategories[i].name {
			t.Errorf("Category[%d].Name = %v, want %v", i, cat.Name, testCategories[i].name)
		}
		if cat.Pattern != testCategories[i].pattern {
			t.Errorf("Category[%d].Pattern = %v, want %v", i, cat.Pattern, testCategories[i].pattern)
		}
	}
}

func TestGetCategory(t *testing.T) {
	db := setupTestDB(t)

	// Insert test category
	_, err := db.Exec("INSERT INTO categories(name, pattern) VALUES(?, ?)", "Test", "test.*")
	if err != nil {
		t.Fatalf("Failed to insert test category: %v", err)
	}

	// Test getting existing category
	category, err := GetCategory(db, int64(1))
	if err != nil {
		t.Errorf("Failed to get category: %v", err)
	}

	if category.Name != "Test" {
		t.Errorf("Category.Name = %v, want Test", category.Name)
	}
	if category.Pattern != "test.*" {
		t.Errorf("Category.Pattern = %v, want test.*", category.Pattern)
	}

	// Test getting non-existent category
	_, err = GetCategory(db, int64(999))
	if err == nil {
		t.Error("Expected error when getting non-existent category, got nil")
	}
}

func TestCreateCategory(t *testing.T) {
	db := setupTestDB(t)

	id, err := CreateCategory(db, "Test", "test.*")
	if err != nil {
		t.Errorf("Failed to create category: %v", err)
	}

	if id != 1 {
		t.Errorf("Expected category ID 1, got %d", id)
	}

	// Verify category was created
	category, err := GetCategory(db, id)
	if err != nil {
		t.Errorf("Failed to get created category: %v", err)
	}

	if category.Name != "Test" {
		t.Errorf("Created category.Name = %v, want Test", category.Name)
	}
	if category.Pattern != "test.*" {
		t.Errorf("Created category.Pattern = %v, want test.*", category.Pattern)
	}
}

func TestDeleteCategories(t *testing.T) {
	db := setupTestDB(t)

	// Create test categories
	_, err := CreateCategory(db, "Food", "restaurant|food")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	_, err = CreateCategory(db, "Transport", "uber|taxi")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	// Drop all categories
	rowsAffected, err := DeleteCategories(db)
	if err != nil {
		t.Errorf("Failed to drop categories: %v", err)
	}

	if rowsAffected != 2 {
		t.Errorf("Expected 2 rows affected, got %d", rowsAffected)
	}

	// Verify categories are deleted
	categories, err := GetCategories(db)
	if err != nil {
		t.Errorf("Failed to get categories after drop: %v", err)
	}
	if len(categories) != 0 {
		t.Errorf("Expected 0 categories after drop, got %d", len(categories))
	}
}

func TestDeleteCategoriesWithExpenses(t *testing.T) {
	db := setupTestDB(t)

	catID, createCategoryErr := CreateCategory(db, "Food", "restaurant")
	if createCategoryErr != nil {
		t.Fatalf("Failed to create test category: %v", createCategoryErr)
	}

	expense := []*Expense{
		{
			Source:      "bank",
			Amount:      -2500,
			Description: "Restaurant dinner",
			Type:        ChargeType,
			Currency:    "EUR",
			CategoryID: sql.NullInt64{
				Int64: catID,
				Valid: true,
			},
		},
	}
	_, insertErr := InsertExpenses(db, expense)
	if insertErr != nil {
		t.Fatalf("Failed to create test expense: %v", insertErr)
	}

	var expensesWithCategoryBefore int
	row := db.QueryRow("SELECT COUNT(*) FROM expenses WHERE category_id IS NOT NULL")
	if beforeQueryErr := row.Scan(&expensesWithCategoryBefore); beforeQueryErr != nil {
		t.Fatalf("Failed to count categorized expenses before drop: %v", beforeQueryErr)
	}
	if expensesWithCategoryBefore != 1 {
		t.Fatalf("Expected 1 categorized expense before drop, got %d", expensesWithCategoryBefore)
	}

	rowsAffected, deleteCategoryErr := DeleteCategories(db)
	if deleteCategoryErr != nil {
		t.Errorf("Failed to drop categories: %v", deleteCategoryErr)
	}

	if rowsAffected != 1 {
		t.Errorf("Expected 1 category deleted, got %d", rowsAffected)
	}

	expenses, getExpensesErr := GetExpenses(db)
	if getExpensesErr != nil {
		t.Errorf("Failed to get expenses after drop: %v", getExpensesErr)
	}
	if len(expenses) != 1 {
		t.Errorf("Expected 1 total expense after drop, got %d", len(expenses))
	}
	if expenses[0].CategoryID.Valid {
		t.Errorf("Expected expense to have null category ID after delete categories")
	}
}

func TestUpdateCategory(t *testing.T) {
	db := setupTestDB(t)

	catID, err := CreateCategory(db, "Food", "restaurant")
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}

	err = UpdateCategory(db, catID, "Dining", "restaurant|dining|food")
	if err != nil {
		t.Errorf("Failed to update category: %v", err)
	}

	category, err := GetCategory(db, catID)
	if err != nil {
		t.Errorf("Failed to get updated category: %v", err)
	}

	if category.Name != "Dining" {
		t.Errorf("Expected category name 'Dining', got '%s'", category.Name)
	}
	if category.Pattern != "restaurant|dining|food" {
		t.Errorf("Expected pattern 'restaurant|dining|food', got '%s'", category.Pattern)
	}
}
