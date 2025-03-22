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
)

type ExpenseType int

const (
	ChargeType ExpenseType = iota
	IncomeType
)

type Expense struct {
	ID          int
	Source      string
	Date        time.Time
	Description string
	Amount      int64
	Type        ExpenseType
	Currency    string
	CategoryID  int

	db *sql.DB
}

func (ex Expense) AmountWithSign() int64 {
	if ex.Type == ChargeType {
		return -(ex.Amount)
	}

	return ex.Amount
}

func (e Expense) Category() (string, error) {
	if e.CategoryID != 0 {
		if e.db == nil {
			fmt.Println("missing db instance on expense instance")
			return "", nil
		}

		c, err := GetCategory(e.db, e.CategoryID)
		if err != nil {
			return "", err
		}

		return c.Name, nil
	}

	return "", nil
}

// content holds our static content.
//
//go:embed templates/*
var content embed.FS

var createTableStatement = `
CREATE TABLE IF NOT EXISTS expenses
(
 id INTEGER PRIMARY KEY,
 source TEXT,
 amount INTEGER NOT NULL,
 description TEXT NOT NULL,
 expense_type INTEGER NOT NULL,
 date INTEGER NOT NULL,
 currency TEXT NOT NULL,
 category_id INTEGER,
 UNIQUE(source, date, description, amount) ON CONFLICT FAIL
) STRICT;
`

func CreateExpenseTable(db *sql.DB) error {
	// Create table
	statement, err := db.Prepare(createTableStatement)
	if err != nil {
		return err
	}

	_, err = statement.Exec()

	return err
}

func DeleteExpenseDB(db *sql.DB) error {
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
	err error
}

func (e ErrInsert) Error() string {
	return fmt.Sprintf("error when trying to insert record on table. err: %v", e.err)
}

func InsertExpenses(db *sql.DB, expenses []*Expense) []error {
	// Insert records
	insertStmt, err := db.Prepare("INSERT INTO expenses(source, amount, description, expense_type, date, currency, category_id) values(?, ?, ?, ?, ?, ?, ?)")

	errors := []error{}

	if err != nil {
		errors = append(errors, err)
		return errors
	}
	for _, expense := range expenses {
		_, err := insertStmt.Exec(expense.Source, expense.Amount, expense.Description, expense.Type, expense.Date.Unix(), expense.Currency, expense.CategoryID)
		if err != nil {
			errors = append(errors, ErrInsert{
				err: err,
			})
		}
	}

	return errors
}

func GetExpenses(db *sql.DB) ([]*Expense, error) {
	rows, err := db.Query("SELECT * FROM expenses")
	if err != nil {
		return []*Expense{}, err
	}

	defer rows.Close()

	expenses := []*Expense{}

	for rows.Next() {
		ex, err := expenseFromRow(db, rows.Scan)

		if err != nil {
			return []*Expense{}, err
		}

		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func UpdateExpenses(db *sql.DB, expenses []*Expense) (int64, error) {
	// Update records
	query := "INSERT OR REPLACE INTO expenses(id, source, amount, description, expense_type, date, currency, category_id) VALUES %s;"
	var buffer = bytes.Buffer{}

	err := renderTemplate(&buffer, "values.tmpl", struct {
		Length   int
		Expenses []*Expense
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

	return result.RowsAffected()
}

func GetExpensesFromDateRange(db *sql.DB, start time.Time, end time.Time) ([]*Expense, error) {
	rows, err := db.Query("SELECT * FROM expenses WHERE date BETWEEN ? and ?", start.Unix(), end.Unix())
	if err != nil {
		return []*Expense{}, err
	}

	defer rows.Close()

	expenses := []*Expense{}

	for rows.Next() {
		ex, err := expenseFromRow(db, rows.Scan)

		if err != nil {
			return []*Expense{}, err
		}

		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func GetExpensesWithoutCategory(db *sql.DB) ([]*Expense, error) {
	rows, err := db.Query("SELECT * FROM expenses WHERE category_id == 0")
	if err != nil {
		return []*Expense{}, err
	}

	defer rows.Close()

	expenses := []*Expense{}

	for rows.Next() {
		ex, err := expenseFromRow(db, rows.Scan)
		if err != nil {
			return []*Expense{}, err
		}
		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func SearchExpenses(db *sql.DB, keyword string) ([]*Expense, error) {
	// search records
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM expenses WHERE description LIKE \"%%%s%%\"", keyword))
	if err != nil {
		return []*Expense{}, err
	}

	defer rows.Close()

	expenses := []*Expense{}

	for rows.Next() {
		ex := &Expense{}
		var id int
		var date int64
		var expenseType int

		if err := rows.Scan(&id, &ex.Amount, &ex.Description, &expenseType, &date, &ex.Currency, &ex.CategoryID); err != nil {
			log.Fatal(err)
		}

		ex.ID = id
		ex.Type = ExpenseType(expenseType)
		ex.Date = time.Unix(date, 0).UTC()
		ex.db = db

		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func SearchExpensesByDescription(db *sql.DB, description string) ([]*Expense, error) {
	// search records
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM expenses WHERE description == \"%s\"", description))
	if err != nil {
		return []*Expense{}, err
	}

	defer rows.Close()

	expenses := []*Expense{}

	for rows.Next() {
		ex, err := expenseFromRow(db, rows.Scan)
		if err != nil {
			return []*Expense{}, err
		}
		expenses = append(expenses, ex)
	}

	return expenses, nil
}

func GetFirstExpense(db *sql.DB) (*Expense, error) {
	row := db.QueryRow("SELECT * FROM expenses  ORDER BY date ASC LIMIT 1")
	ex, err := expenseFromRow(db, row.Scan)

	if err != nil {
		return &Expense{}, err
	}

	return ex, nil
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

func expenseFromRow(db *sql.DB, scan func(dest ...any) error) (*Expense, error) {
	ex := &Expense{}
	var id int
	var date int64
	var expenseType int

	if err := scan(&id, &ex.Source, &ex.Amount, &ex.Description, &expenseType, &date, &ex.Currency, &ex.CategoryID); err != nil {
		return ex, err
	}

	ex.ID = id
	ex.Type = ExpenseType(expenseType)
	ex.Date = time.Unix(date, 0).UTC()
	ex.db = db

	return ex, nil
}
