package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Category struct {
	ID      int
	Name    string
	Pattern string
}

func GetCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.QueryContext(context.Background(), "SELECT * FROM categories")
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
	row := db.QueryRowContext(context.Background(), "SELECT * FROM categories WHERE id=$1", categoryID)
	return categoryFromRow(row.Scan)
}

func UpdateCategory(db *sql.DB, categoryID int, name, pattern string) error {
	_, err := db.ExecContext(context.Background(),
		"UPDATE categories SET name = ?, pattern = ? WHERE id = ?;",
		name,
		pattern,
		categoryID,
	)
	return err
}

func CreateCategory(db *sql.DB, name, pattern string) (int64, error) {
	result, err := db.ExecContext(context.Background(),
		"INSERT INTO categories(name, pattern) values(?, ?)", name, pattern)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func DeleteCategories(db *sql.DB) (int64, error) {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return 0, err
	}

	_, err = tx.ExecContext(context.Background(),
		"UPDATE expenses SET category_id = NULL WHERE category_id IS NOT NULL")
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("failed to uncategorize expenses: %w", err)
	}

	result, err := tx.ExecContext(context.Background(), "DELETE FROM categories")
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("failed to delete categories: %w", err)
	}

	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result.RowsAffected()
}

func categoryFromRow(scan func(dest ...any) error) (Category, error) {
	cat := Category{}
	var id int

	if err := scan(&id, &cat.Name, &cat.Pattern); err != nil {
		return cat, err
	}

	cat.ID = id

	return cat, nil
}
