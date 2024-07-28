package db

import (
	"database/sql"
	"log"
	"time"

	"github.com/GustavoCaso/sandbox/go/moneyTracker/pkg/expense"
	_ "github.com/mattn/go-sqlite3"
)

func GetOrCreateExpenseDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "expenses.db")
	if err != nil {
		return nil, err
	}

	// Create table
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS expenses (id INTEGER PRIMARY KEY, amount INTEGER NOT NULL, decimal INTEGER NOT NULL,  description TEXT NOT NULL, expense_type INTEGER NOT NULL, date INTEGER NOT NULL, currency TEXT NOT NULL, category TEXT NOT NULL) STRICT;")
	if err != nil {
		return nil, err
	}

	_, err = statement.Exec()

	if err != nil {
		return nil, err
	}

	return db, nil
}

func InsertExpenses(db *sql.DB, expenses []expense.Expense) error {
	// Insert records
	insertStmt, err := db.Prepare("INSERT INTO expenses(amount, decimal, description, expense_type, date, currency, category) values(?, ?, ?, ?, ?, ?, ?)")

	if err != nil {
		return err
	}
	for _, expense := range expenses {
		_, err := insertStmt.Exec(expense.Amount, expense.Decimal, expense.Description, expense.Type, expense.Date.Unix(), expense.Currency, expense.Category)
		if err != nil {
			return err
		}
	}

	return nil
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

		if err := rows.Scan(&id, &ex.Amount, &ex.Decimal, &ex.Description, &expenseType, &date, &ex.Currency, &ex.Category); err != nil {
			log.Fatal(err)
		}

		ex.Type = expense.ExpenseType(expenseType)
		ex.Date = time.Unix(date, 0).UTC()

		expenses = append(expenses, ex)
	}

	return expenses, nil
}
