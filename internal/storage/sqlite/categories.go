package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func (s *sqliteStorage) GetCategories(ctx context.Context) ([]storage.Category, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT * FROM categories")
	if err != nil {
		return []storage.Category{}, err
	}

	if rows.Err() != nil {
		return []storage.Category{}, rows.Err()
	}

	defer rows.Close()

	categories := []storage.Category{}

	for rows.Next() {
		category, categoryErr := categoryFromRow(rows.Scan)

		if categoryErr != nil {
			return categories, categoryErr
		}

		categories = append(categories, category)
	}

	return categories, nil
}

func (s *sqliteStorage) GetCategory(ctx context.Context, categoryID int64) (storage.Category, error) {
	row := s.db.QueryRowContext(ctx, "SELECT * FROM categories WHERE id=?", categoryID)
	return categoryFromRow(row.Scan)
}

func (s *sqliteStorage) GetExcludeCategory(ctx context.Context) (storage.Category, error) {
	row := s.db.QueryRowContext(ctx, "SELECT * FROM categories WHERE name=?", storage.ExcludeCategory)
	return categoryFromRow(row.Scan)
}

func (s *sqliteStorage) UpdateCategory(ctx context.Context, categoryID int64, name, pattern string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE categories SET name = ?, pattern = ? WHERE id = ?;",
		name,
		pattern,
		categoryID,
	)
	return err
}

func (s *sqliteStorage) CreateCategory(ctx context.Context, name, pattern string) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO categories(name, pattern) values(?, ?)", name, pattern)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (s *sqliteStorage) DeleteCategories(ctx context.Context) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	excludeRow := tx.QueryRowContext(ctx, "SELECT * FROM categories WHERE name=?", storage.ExcludeCategory)
	excludeCategory, excludeErr := categoryFromRow(excludeRow.Scan)
	if excludeErr != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("failed to fetch exclude category: %w", excludeErr)
	}

	_, err = tx.ExecContext(
		ctx,
		"UPDATE expenses SET category_id = NULL WHERE category_id IS NOT NULL AND category_id IS NOT ?",
		excludeCategory.ID(),
	)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("failed to uncategorize expenses: %w", err)
	}

	result, err := tx.ExecContext(ctx, "DELETE FROM categories WHERE id IS NOT ?", excludeCategory.ID())
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

func (s *sqliteStorage) DeleteCategory(ctx context.Context, id int64) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	_, err = tx.ExecContext(
		ctx,
		"UPDATE expenses SET category_id = NULL WHERE category_id IS ?",
		id,
	)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("failed to uncategorize expenses: %w", err)
	}

	result, err := tx.ExecContext(ctx, "DELETE FROM categories WHERE id IS ?", id)
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

func categoryFromRow(scan func(dest ...any) error) (storage.Category, error) {
	// Use the Category type from expenses.go
	var id int64
	var name, pattern string

	if err := scan(&id, &name, &pattern); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &storage.NotFoundError{}
		}
		return nil, err
	}

	return storage.NewCategory(id, name, pattern), nil
}
