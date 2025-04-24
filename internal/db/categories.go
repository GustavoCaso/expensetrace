package db

import (
	"database/sql"
	"errors"

	"github.com/GustavoCaso/expensetrace/internal/config"

	sqlite3 "github.com/mattn/go-sqlite3"
)

type CategoryType int

const (
	ExpenseCategoryType CategoryType = iota
	IncomeCategoryType
)

type Category struct {
	ID      int
	Name    string
	Pattern string
	Type    CategoryType
}

func PopulateCategoriesFromConfig(db *sql.DB, conf *config.Config) error {
	// Insert records
	insertStmt, err := db.Prepare("INSERT INTO categories(name, pattern, type) values(?, ?, ?)")

	var e error

	if err != nil {
		return err
	}
	defer insertStmt.Close()

	for _, category := range conf.Categories.Expense {
		_, err = insertStmt.Exec(category.Name, category.Pattern, ExpenseCategoryType)

		if err != nil {
			if errors.Is(err, sqlite3.Error{Code: sqlite3.ErrConstraint}) {
				e = errors.Join(e, InsertError{
					err: err,
				})
			}
		}
	}

	for _, category := range conf.Categories.Income {
		_, err = insertStmt.Exec(category.Name, category.Pattern, IncomeCategoryType)

		if err != nil {
			if errors.Is(err, sqlite3.Error{Code: sqlite3.ErrConstraint}) {
				e = errors.Join(e, InsertError{
					err: err,
				})
			}
		}
	}

	return e
}

func GetCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.Query("SELECT * FROM categories")
	if err != nil {
		return []Category{}, err
	}

	if rows.Err() != nil {
		return []Category{}, rows.Err()
	}

	defer rows.Close()

	categories := []Category{}

	for rows.Next() {
		category, categoryErr := categoryFromRow(rows.Scan)

		if categoryErr != nil {
			return categories, categoryErr
		}

		categories = append(categories, category)
	}

	return categories, nil
}

func GetCategory(db *sql.DB, categoryID int64) (Category, error) {
	row := db.QueryRow("SELECT * FROM categories WHERE id=$1", categoryID)
	return categoryFromRow(row.Scan)
}

func UpdateCategory(db *sql.DB, categoryID int, name, pattern string, categoryType CategoryType) error {
	_, err := db.Exec(
		"UPDATE categories SET name = ?, pattern = ?, type = ? WHERE id = ?;",
		name,
		pattern,
		categoryType,
		categoryID,
	)
	return err
}

func CreateCategory(db *sql.DB, name, pattern string, categoryType CategoryType) (int64, error) {
	result, err := db.Exec("INSERT INTO categories(name, pattern, type) values(?, ?, ?)", name, pattern, categoryType)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func categoryFromRow(scan func(dest ...any) error) (Category, error) {
	cat := Category{}
	var id int
	var categoryType int

	if err := scan(&id, &cat.Name, &cat.Pattern, &categoryType); err != nil {
		return cat, err
	}

	cat.ID = id
	cat.Type = CategoryType(categoryType)

	return cat, nil
}
