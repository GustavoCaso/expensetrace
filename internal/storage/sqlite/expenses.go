package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"path"
	"text/template"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/filter"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func convertToTemplateExpenses(userID int64, expenses []storage.Expense) []*templateExpense {
	templateExpenses := make([]*templateExpense, len(expenses))
	for i, exp := range expenses {
		categoryID := sql.NullInt64{}
		if exp.CategoryID() != nil {
			categoryID = sql.NullInt64{Int64: *exp.CategoryID(), Valid: true}
		}

		templateExpenses[i] = &templateExpense{
			ID:          int(exp.ID()),
			Source:      exp.Source(),
			Date:        exp.Date(),
			Description: exp.Description(),
			Amount:      exp.Amount(),
			Type:        exp.Type(),
			Currency:    exp.Currency(),
			CategoryID:  categoryID,
			UserID:      userID,
		}
	}
	return templateExpenses
}

type templateExpense struct {
	ID          int
	Source      string
	Date        time.Time
	Description string
	Amount      int64
	Type        storage.ExpenseType
	Currency    string
	CategoryID  sql.NullInt64
	UserID      int64
}

// content holds our static content.
//
//go:embed templates/*
var content embed.FS

func (s *sqliteStorage) GetExpenseByID(ctx context.Context, userID, id int64) (storage.Expense, error) {
	row := s.db.QueryRowContext(ctx, "SELECT * FROM expenses WHERE id = ? AND user_id = ?", id, userID)
	return expenseFromRow(row.Scan)
}

func (s *sqliteStorage) UpdateExpense(ctx context.Context, userID int64, expense storage.Expense) (int64, error) {
	categoryID := sql.NullInt64{}
	if expense.CategoryID() != nil {
		categoryID = sql.NullInt64{Int64: *expense.CategoryID(), Valid: true}
	}

	r, err := s.db.ExecContext(ctx,
		`UPDATE expenses SET source = ?, amount = ?, description = ?,
		 expense_type = ?, date = ?, currency = ?, category_id = ?
		 WHERE id = ? AND user_id = ?`,
		expense.Source(), expense.Amount(), expense.Description(),
		expense.Type(), expense.Date().Unix(), expense.Currency(),
		categoryID, expense.ID(), userID)
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}

func (s *sqliteStorage) DeleteExpense(ctx context.Context, userID, id int64) (int64, error) {
	r, err := s.db.ExecContext(ctx,
		"DELETE FROM expenses WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}

func (s *sqliteStorage) InsertExpenses(ctx context.Context, userID int64, expenses []storage.Expense) (int64, error) {
	if len(expenses) == 0 {
		return 0, nil
	}

	// Convert to internal template-compatible type
	templateExpenses := convertToTemplateExpenses(userID, expenses)

	// Insert records
	query := "INSERT OR IGNORE INTO expenses(source, amount, description, expense_type, date, currency, category_id, user_id) VALUES %s;"
	var buffer = bytes.Buffer{}

	err := s.renderTemplate(&buffer, "expenses/insert.tmpl", struct {
		Length   int
		Expenses []*templateExpense
	}{
		// Inside the template we iterate over expenses, the index starts at 0
		Length:   len(templateExpenses) - 1,
		Expenses: templateExpenses,
	})

	if err != nil {
		return 0, err
	}

	formattedQuery := fmt.Sprintf(query, buffer.String())

	result, err := s.db.ExecContext(ctx, formattedQuery)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (s *sqliteStorage) GetExpenses(ctx context.Context, userID int64) ([]storage.Expense, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT * FROM expenses WHERE expense_type = 0 AND user_id = ?", userID)
	if err != nil {
		return []storage.Expense{}, err
	}

	return extractExpensesFromRows(rows)
}

func (s *sqliteStorage) GetAllExpenseTypes(ctx context.Context, userID int64) ([]storage.Expense, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT * FROM expenses WHERE user_id = ?", userID)
	if err != nil {
		return []storage.Expense{}, err
	}

	return extractExpensesFromRows(rows)
}

func (s *sqliteStorage) UpdateExpenses(ctx context.Context, userID int64, expenses []storage.Expense) (int64, error) {
	if len(expenses) == 0 {
		return 0, nil
	}

	// Convert to internal template-compatible type
	templateExpenses := convertToTemplateExpenses(userID, expenses)

	// Update records
	query := "INSERT OR REPLACE INTO expenses(id, source, amount, description, expense_type, date, currency, category_id, user_id) VALUES %s;"
	var buffer = bytes.Buffer{}

	err := s.renderTemplate(&buffer, "expenses/updates.tmpl", struct {
		Length   int
		Expenses []*templateExpense
	}{
		// Inside the template we iterate over expenses, the index starts at 0
		Length:   len(templateExpenses) - 1,
		Expenses: templateExpenses,
	})

	if err != nil {
		return 0, err
	}

	formattedQuery := fmt.Sprintf(query, buffer.String())

	result, err := s.db.ExecContext(ctx, formattedQuery)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (s *sqliteStorage) GetExpensesFromDateRange(
	ctx context.Context,
	userID int64,
	start time.Time,
	end time.Time,
) ([]storage.Expense, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT * FROM expenses WHERE date BETWEEN ? and ? AND user_id = ?", start.Unix(), end.Unix(), userID)
	if err != nil {
		return []storage.Expense{}, err
	}

	return extractExpensesFromRows(rows)
}

func (s *sqliteStorage) GetExpensesWithoutCategory(ctx context.Context, userID int64) ([]storage.Expense, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT * FROM expenses WHERE category_id IS NULL AND expense_type = 0 AND user_id = ?", userID)
	if err != nil {
		return []storage.Expense{}, err
	}

	return extractExpensesFromRows(rows)
}

func (s *sqliteStorage) SearchExpenses(ctx context.Context, userID int64, keyword string) ([]storage.Expense, error) {
	// Use parameterized query to prevent SQL injection
	rows, err := s.db.QueryContext(ctx,
		"SELECT * FROM expenses WHERE description LIKE ? AND user_id = ?", "%"+keyword+"%", userID)
	if err != nil {
		return []storage.Expense{}, err
	}

	return extractExpensesFromRows(rows)
}

func (s *sqliteStorage) SearchExpensesByDescription(
	ctx context.Context,
	userID int64,
	description string,
) ([]storage.Expense, error) {
	// Use parameterized query to prevent SQL injection
	rows, err := s.db.QueryContext(ctx,
		"SELECT * FROM expenses WHERE description = ? AND user_id = ?", description, userID)
	if err != nil {
		return []storage.Expense{}, err
	}

	return extractExpensesFromRows(rows)
}

func (s *sqliteStorage) GetExpensesWithoutCategoryWithQuery(
	ctx context.Context,
	userID int64,
	keyword string,
) ([]storage.Expense, error) {
	rows, err := s.db.QueryContext(
		ctx,
		"SELECT * FROM expenses WHERE category_id IS NULL AND expense_type = 0 AND description LIKE ? AND user_id = ?",
		"%"+keyword+"%",
		userID,
	)
	if err != nil {
		return []storage.Expense{}, err
	}

	return extractExpensesFromRows(rows)
}

func (s *sqliteStorage) GetFirstExpense(ctx context.Context, userID int64) (storage.Expense, error) {
	row := s.db.QueryRowContext(ctx, "SELECT * FROM expenses WHERE user_id = ? ORDER BY date ASC LIMIT 1", userID)
	return expenseFromRow(row.Scan)
}

func (s *sqliteStorage) GetExpensesByCategory(
	ctx context.Context,
	userID, categoryID int64,
) ([]storage.Expense, error) {
	rows, err := s.db.QueryContext(
		ctx,
		"SELECT * FROM expenses WHERE category_id = ? AND user_id = ?",
		categoryID,
		userID,
	)
	if err != nil {
		return []storage.Expense{}, err
	}

	return extractExpensesFromRows(rows)
}

func (s *sqliteStorage) GetExpensesFiltered(
	ctx context.Context,
	userID int64,
	expFilter *filter.ExpenseFilter,
	sort *filter.SortOptions,
) ([]storage.Expense, error) {
	query := "SELECT * FROM expenses WHERE user_id = ?"
	args := []interface{}{userID}

	// Add filters dynamically
	if expFilter.Description != nil {
		query += " AND description LIKE ?"
		args = append(args, "%"+*expFilter.Description+"%")
	}

	if expFilter.Source != nil {
		query += " AND source LIKE ?"
		args = append(args, "%"+*expFilter.Source+"%")
	}

	if expFilter.AmountMin != nil {
		query += " AND amount >= ?"
		args = append(args, *expFilter.AmountMin)
	}

	if expFilter.AmountMax != nil {
		query += " AND amount <= ?"
		args = append(args, *expFilter.AmountMax)
	}

	if expFilter.DateFrom != nil {
		query += " AND date >= ?"
		args = append(args, expFilter.DateFrom.Unix())
	}

	if expFilter.DateTo != nil {
		query += " AND date <= ?"
		args = append(args, expFilter.DateTo.Unix())
	}

	// Add sorting
	query += fmt.Sprintf(" ORDER BY %s %s", sort.Field, sort.Direction)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return extractExpensesFromRows(rows)
}

func (s *sqliteStorage) renderTemplate(out io.Writer, templateName string, value any) error {
	tmpl, err := content.ReadFile(path.Join("templates", templateName))
	if err != nil {
		return err
	}
	t := template.Must(template.New(templateName).Parse(string(tmpl)))
	err = t.Execute(out, value)
	if err != nil {
		return err
	}

	return nil
}

func extractExpensesFromRows(rows *sql.Rows) ([]storage.Expense, error) {
	if rows.Err() != nil {
		return []storage.Expense{}, rows.Err()
	}

	defer rows.Close()

	expenses := []storage.Expense{}

	for rows.Next() {
		ex, expenseErr := expenseFromRow(rows.Scan)

		if expenseErr != nil {
			return []storage.Expense{}, expenseErr
		}

		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func expenseFromRow(scan func(dest ...any) error) (storage.Expense, error) {
	var id int64
	var source string
	var amount int64
	var description string
	var expenseType int
	var date int64
	var currency string
	var categoryID sql.NullInt64
	var userID int64

	if err := scan(&id, &source, &amount, &description, &expenseType, &date, &currency, &categoryID, &userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &storage.NotFoundError{}
		}
		return nil, err
	}

	var catID *int64
	if categoryID.Valid {
		catID = &categoryID.Int64
	} else {
		catID = nil
	}

	return storage.NewExpense(
		id,
		source,
		description,
		currency,
		amount,
		time.Unix(date, 0).UTC(),
		storage.ExpenseType(expenseType),
		catID,
	), nil
}
