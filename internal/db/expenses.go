package db

import (
	"bytes"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"log"
	"path"
	"text/template"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/expense"
	_ "github.com/mattn/go-sqlite3"
)

// content holds our static content.
//
//go:embed templates/*
var content embed.FS

var createTableStatement = `
CREATE TABLE IF NOT EXISTS expenses
(
 id INTEGER PRIMARY KEY,
 amount INTEGER NOT NULL,
 description TEXT NOT NULL,
 expense_type INTEGER NOT NULL,
 date INTEGER NOT NULL,
 currency TEXT NOT NULL,
 category TEXT NOT NULL,
 UNIQUE(date, description, amount) ON CONFLICT FAIL
) STRICT;
`

func GetOrCreateExpenseDB(dbsource string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbsource)
	if err != nil {
		return nil, err
	}

	// Create table
	statement, err := db.Prepare(createTableStatement)
	if err != nil {
		return nil, err
	}

	_, err = statement.Exec()

	if err != nil {
		return nil, err
	}

	return db, nil
}

func DeleteExpenseDB(dbsource string) error {
	db, err := sql.Open("sqlite3", dbsource)
	if err != nil {
		return err
	}

	// drop table
	statement, err := db.Prepare("DROP TABLE IF EXISTS expenses;")
	if err != nil {
		return err
	}

	_, err = statement.Exec()

	if err != nil {
		return err
	}

	return nil
}

type ErrInsert struct {
	expense expense.Expense
	err     error
}

func (e ErrInsert) Error() string {
	return fmt.Sprintf("error when trying to insert expense\n expense: %+v\n err: %v", e.expense, e.err)
}

func InsertExpenses(db *sql.DB, expenses []expense.Expense) []error {
	// Insert records
	insertStmt, err := db.Prepare("INSERT INTO expenses(amount, description, expense_type, date, currency, category) values(?, ?, ?, ?, ?, ?)")

	errors := []error{}

	if err != nil {
		errors = append(errors, err)
		return errors
	}
	for _, expense := range expenses {
		_, err := insertStmt.Exec(expense.Amount, expense.Description, expense.Type, expense.Date.Unix(), expense.Currency, expense.Category)
		if err != nil {
			errors = append(errors, ErrInsert{
				expense: expense,
				err:     err,
			})
		}
	}

	return errors
}

func UpdateExpenses(db *sql.DB, expenses []expense.Expense) (int64, error) {
	// Update records
	query := "INSERT OR REPLACE INTO expenses(id, amount, description, expense_type, date, currency, category) VALUES %s;"
	var buffer = bytes.Buffer{}

	err := renderTemplate(&buffer, "values.tmpl", struct {
		Length   int
		Expenses []expense.Expense
	}{
		// Inside the template we itarte over expenses, the index starts at 0
		Length:   len(expenses) - 1,
		Expenses: expenses,
	})

	if err != nil {
		return 0, err
	}

	formattedQuery := fmt.Sprintf(query, buffer.String())

	result, err := db.Exec(formattedQuery)
	if err != nil {
		return 0, err
	}

	count, _ := result.RowsAffected()

	return count, nil
}

func GetExpensesFromDateRange(db *sql.DB, start time.Time, end time.Time) ([]expense.Expense, error) {
	rows, err := db.Query("SELECT * FROM expenses WHERE date BETWEEN ? and ?", start.Unix(), end.Unix())
	if err != nil {
		return []expense.Expense{}, err
	}

	defer rows.Close()

	expenses := []expense.Expense{}

	for rows.Next() {
		var ex expense.Expense
		var id int
		var date int64
		var expenseType int

		if err := rows.Scan(&id, &ex.Amount, &ex.Description, &expenseType, &date, &ex.Currency, &ex.Category); err != nil {
			log.Fatal(err)
		}

		ex.ID = id
		ex.Type = expense.ExpenseType(expenseType)
		ex.Date = time.Unix(date, 0).UTC()

		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func GetExpensesWithoutCategory(db *sql.DB) ([]expense.Expense, error) {
	rows, err := db.Query("SELECT * FROM expenses WHERE category == \"\"")
	if err != nil {
		return []expense.Expense{}, err
	}

	defer rows.Close()

	expenses := []expense.Expense{}

	for rows.Next() {
		var ex expense.Expense
		var id int
		var date int64
		var expenseType int

		if err := rows.Scan(&id, &ex.Amount, &ex.Description, &expenseType, &date, &ex.Currency, &ex.Category); err != nil {
			log.Fatal(err)
		}

		ex.ID = id
		ex.Type = expense.ExpenseType(expenseType)
		ex.Date = time.Unix(date, 0).UTC()

		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func renderTemplate(out io.Writer, templateName string, value interface{}) error {
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
