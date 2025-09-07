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

	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func convertToTemplateExpenses(expenses []storage.Expense) []*templateExpense {
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
}

// content holds our static content.
//
//go:embed templates/*
var content embed.FS

func (s *sqliteStorage) GetExpense() (storage.Expense, error) {
	row := s.db.QueryRowContext(context.Background(), "SELECT * FROM expenses LIMIT 1")
	return s.expenseFromRow(row.Scan)
}

func (s *sqliteStorage) GetExpenseByID(id int64) (storage.Expense, error) {
	row := s.db.QueryRowContext(context.Background(), "SELECT * FROM expenses WHERE id = ?", id)
	return s.expenseFromRow(row.Scan)
}

func (s *sqliteStorage) UpdateExpense(expense storage.Expense) (int64, error) {
	categoryID := sql.NullInt64{}
	if expense.CategoryID() != nil {
		categoryID = sql.NullInt64{Int64: *expense.CategoryID(), Valid: true}
	}

	r, err := s.db.ExecContext(context.Background(),
		`UPDATE expenses SET source = ?, amount = ?, description = ?, 
		 expense_type = ?, date = ?, currency = ?, category_id = ? 
		 WHERE id = ?`,
		expense.Source(), expense.Amount(), expense.Description(),
		expense.Type(), expense.Date().Unix(), expense.Currency(),
		categoryID, expense.ID())
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}

func (s *sqliteStorage) DeleteExpense(id int64) (int64, error) {
	r, err := s.db.ExecContext(context.Background(),
		"DELETE FROM expenses WHERE id = ?", id)
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}

func (s *sqliteStorage) InsertExpenses(expenses []storage.Expense) (int64, error) {
	if len(expenses) == 0 {
		return 0, nil
	}

	// Convert to internal template-compatible type
	templateExpenses := convertToTemplateExpenses(expenses)

	// Insert records
	query := "INSERT OR IGNORE INTO expenses(source, amount, description, expense_type, date, currency, category_id) VALUES %s;"
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

	result, err := s.db.ExecContext(context.Background(), formattedQuery)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (s *sqliteStorage) GetExpenses() ([]storage.Expense, error) {
	rows, err := s.db.QueryContext(context.Background(), "SELECT * FROM expenses")
	if err != nil {
		return []storage.Expense{}, err
	}

	if rows.Err() != nil {
		return []storage.Expense{}, rows.Err()
	}

	defer rows.Close()

	expenses := []storage.Expense{}

	for rows.Next() {
		ex, expenseErr := s.expenseFromRow(rows.Scan)

		if expenseErr != nil {
			return []storage.Expense{}, expenseErr
		}

		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func (s *sqliteStorage) UpdateExpenses(expenses []storage.Expense) (int64, error) {
	if len(expenses) == 0 {
		return 0, nil
	}

	// Convert to internal template-compatible type
	templateExpenses := convertToTemplateExpenses(expenses)

	// Update records
	query := "INSERT OR REPLACE INTO expenses(id, source, amount, description, expense_type, date, currency, category_id) VALUES %s;"
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

	result, err := s.db.ExecContext(context.Background(), formattedQuery)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (s *sqliteStorage) GetExpensesFromDateRange(start time.Time, end time.Time) ([]storage.Expense, error) {
	rows, err := s.db.QueryContext(context.Background(),
		"SELECT * FROM expenses WHERE date BETWEEN ? and ?", start.Unix(), end.Unix())
	if err != nil {
		return []storage.Expense{}, err
	}

	if rows.Err() != nil {
		return []storage.Expense{}, rows.Err()
	}

	defer rows.Close()

	expenses := []storage.Expense{}

	for rows.Next() {
		ex, expenseErr := s.expenseFromRow(rows.Scan)

		if expenseErr != nil {
			return []storage.Expense{}, expenseErr
		}

		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func (s *sqliteStorage) GetExpensesWithoutCategory() ([]storage.Expense, error) {
	rows, err := s.db.QueryContext(context.Background(),
		"SELECT * FROM expenses WHERE category_id IS NULL AND expense_type = 0")
	if err != nil {
		return []storage.Expense{}, err
	}

	if rows.Err() != nil {
		return []storage.Expense{}, rows.Err()
	}

	defer rows.Close()

	expenses := []storage.Expense{}

	for rows.Next() {
		ex, expenseErr := s.expenseFromRow(rows.Scan)
		if expenseErr != nil {
			return []storage.Expense{}, expenseErr
		}
		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func (s *sqliteStorage) SearchExpenses(keyword string) ([]storage.Expense, error) {
	// Use parameterized query to prevent SQL injection
	rows, err := s.db.QueryContext(context.Background(),
		"SELECT * FROM expenses WHERE description LIKE ?", "%"+keyword+"%")
	if err != nil {
		return []storage.Expense{}, err
	}

	if rows.Err() != nil {
		return []storage.Expense{}, rows.Err()
	}

	defer rows.Close()

	expenses := []storage.Expense{}

	for rows.Next() {
		ex, expenseErr := s.expenseFromRow(rows.Scan)

		if expenseErr != nil {
			return []storage.Expense{}, expenseErr
		}

		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func (s *sqliteStorage) SearchExpensesByDescription(description string) ([]storage.Expense, error) {
	// Use parameterized query to prevent SQL injection
	rows, err := s.db.QueryContext(context.Background(),
		"SELECT * FROM expenses WHERE description = ?", description)
	if err != nil {
		return []storage.Expense{}, err
	}

	if rows.Err() != nil {
		return []storage.Expense{}, rows.Err()
	}

	defer rows.Close()

	expenses := []storage.Expense{}

	for rows.Next() {
		ex, expenseErr := s.expenseFromRow(rows.Scan)
		if expenseErr != nil {
			return []storage.Expense{}, expenseErr
		}
		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func (s *sqliteStorage) GetFirstExpense() (storage.Expense, error) {
	row := s.db.QueryRowContext(context.Background(), "SELECT * FROM expenses ORDER BY date ASC LIMIT 1")
	return s.expenseFromRow(row.Scan)
}

func (s *sqliteStorage) GetExpensesByCategory(categoryID int64) ([]storage.Expense, error) {
	rows, err := s.db.QueryContext(context.Background(), "SELECT * FROM expenses WHERE category_id = ?", categoryID)
	if err != nil {
		return []storage.Expense{}, err
	}

	if rows.Err() != nil {
		return []storage.Expense{}, rows.Err()
	}

	defer rows.Close()

	expenses := []storage.Expense{}

	for rows.Next() {
		ex, expenseErr := s.expenseFromRow(rows.Scan)
		if expenseErr != nil {
			return []storage.Expense{}, expenseErr
		}
		expenses = append(expenses, ex)
	}

	return expenses, nil
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

func (s *sqliteStorage) expenseFromRow(scan func(dest ...any) error) (storage.Expense, error) {
	var id int64
	var source string
	var amount int64
	var description string
	var expenseType int
	var date int64
	var currency string
	var categoryID sql.NullInt64

	if err := scan(&id, &source, &amount, &description, &expenseType, &date, &currency, &categoryID); err != nil {
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
