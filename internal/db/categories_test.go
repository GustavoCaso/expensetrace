package db

import (
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/config"
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

func TestPopulateCategoriesFromConfig(t *testing.T) {
	db := setupTestDB(t)

	conf := &config.Config{
		Categories: []config.Category{
			{Name: "Food", Pattern: "restaurant|food"},
			{Name: "Transport", Pattern: "uber|taxi"},
		},
	}

	err := PopulateCategoriesFromConfig(db, conf)
	if err != nil {
		t.Errorf("Failed to populate categories: %v", err)
	}

	// Verify categories were inserted
	categories, err := GetCategories(db)
	if err != nil {
		t.Errorf("Failed to get categories: %v", err)
	}

	if len(categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(categories))
	}

	// Verify category contents
	for i, cat := range categories {
		if cat.Name != conf.Categories[i].Name {
			t.Errorf("Category[%d].Name = %v, want %v", i, cat.Name, conf.Categories[i].Name)
		}
		if cat.Pattern != conf.Categories[i].Pattern {
			t.Errorf("Category[%d].Pattern = %v, want %v", i, cat.Pattern, conf.Categories[i].Pattern)
		}
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
