package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/GustavoCaso/expensetrace/domain"
)

func (s *sqliteStorage) GetCategories(ctx context.Context, userID int64) ([]domain.Category, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+categoryColumns+" FROM categories WHERE user_id = ?", userID)
	if err != nil {
		return []domain.Category{}, err
	}

	if rows.Err() != nil {
		return []domain.Category{}, rows.Err()
	}

	defer rows.Close()

	categories := []domain.Category{}

	for rows.Next() {
		category, categoryErr := categoryFromRow(rows.Scan)

		if categoryErr != nil {
			return categories, categoryErr
		}

		categories = append(categories, category)
	}

	return categories, nil
}

func (s *sqliteStorage) GetCategory(ctx context.Context, userID, categoryID int64) (domain.Category, error) {
	row := s.db.QueryRowContext(
		ctx,
		"SELECT "+categoryColumns+" FROM categories WHERE id=? AND user_id = ?",
		categoryID,
		userID,
	)
	return categoryFromRow(row.Scan)
}

func (s *sqliteStorage) GetExcludeCategory(ctx context.Context, userID int64) (domain.Category, error) {
	row := s.db.QueryRowContext(
		ctx,
		"SELECT "+categoryColumns+" FROM categories WHERE name=? AND user_id = ?",
		domain.ExcludeCategory,
		userID,
	)
	return categoryFromRow(row.Scan)
}

func (s *sqliteStorage) UpdateCategory(
	ctx context.Context,
	userID, categoryID int64,
	name, pattern string,
	monthlyBudget int64,
	categoryType domain.CategoryType,
) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE categories SET name = ?, pattern = ?, monthly_budget = ?, type = ? WHERE id = ? AND user_id = ?;",
		name,
		pattern,
		monthlyBudget,
		categoryType,
		categoryID,
		userID,
	)
	return err
}

func (s *sqliteStorage) CreateCategory(
	ctx context.Context,
	userID int64,
	name, pattern string,
	monthlyBudget int64,
	categoryType domain.CategoryType,
) (int64, error) {
	result, err := s.db.ExecContext(
		ctx,
		"INSERT INTO categories(name, pattern, user_id, monthly_budget, type) values(?, ?, ?, ?, ?)",
		name,
		pattern,
		userID,
		monthlyBudget,
		categoryType,
	)

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (s *sqliteStorage) DeleteCategories(ctx context.Context, userID int64) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	excludeRow := tx.QueryRowContext(
		ctx,
		"SELECT "+categoryColumns+" FROM categories WHERE name=? AND user_id = ?",
		domain.ExcludeCategory,
		userID,
	)
	excludeCategory, excludeErr := categoryFromRow(excludeRow.Scan)
	if excludeErr != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("failed to fetch exclude category: %w", excludeErr)
	}

	_, err = tx.ExecContext(
		ctx,
		"UPDATE expenses SET category_id = NULL WHERE category_id IS NOT NULL AND category_id IS NOT ? AND user_id = ?",
		excludeCategory.ID(),
		userID,
	)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("failed to uncategorize expenses: %w", err)
	}

	result, err := tx.ExecContext(
		ctx,
		"DELETE FROM categories WHERE id IS NOT ? AND user_id = ?",
		excludeCategory.ID(),
		userID,
	)
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

func (s *sqliteStorage) DeleteCategory(ctx context.Context, userID, categoryID int64) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	_, err = tx.ExecContext(
		ctx,
		"UPDATE expenses SET category_id = NULL WHERE category_id IS ? AND user_id = ?",
		categoryID,
		userID,
	)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("failed to uncategorize expenses: %w", err)
	}

	result, err := tx.ExecContext(ctx, "DELETE FROM categories WHERE id IS ? AND user_id = ?", categoryID, userID)
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

// categoryColumns lists the columns categoryFromRow scans, in order.
const categoryColumns = "id, name, pattern, monthly_budget, type"

func categoryFromRow(scan func(dest ...any) error) (domain.Category, error) {
	var id int64
	var name, pattern string
	var monthlyBudget int64
	var categoryType domain.CategoryType

	if err := scan(&id, &name, &pattern, &monthlyBudget, &categoryType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{}
		}
		return nil, err
	}

	return domain.NewCategory(id, name, pattern, monthlyBudget, categoryType), nil
}
