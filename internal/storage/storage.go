package storage

import (
	"time"

	"github.com/GustavoCaso/expensetrace/internal/logger"
)

type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "record not found"
}

type Category interface {
	ID() int64
	Name() string
	Pattern() string
}

type category struct {
	id      int64
	name    string
	pattern string
}

func (c category) ID() int64 {
	return c.id
}

func (c category) Name() string {
	return c.name
}

func (c category) Pattern() string {
	return c.pattern
}

func NewCategory(id int64, name, pattern string) Category {
	return category{
		id:      id,
		name:    name,
		pattern: pattern,
	}
}

type ExpenseType int

const (
	ChargeType ExpenseType = iota
	IncomeType
)

type Expense interface {
	ID() int64
	Source() string
	Date() time.Time
	Description() string
	Amount() int64
	Type() ExpenseType
	Currency() string
	CategoryID() *int64
}

type expense struct {
	id          int64
	source      string
	date        time.Time
	description string
	amount      int64
	expenseType ExpenseType
	currency    string
	categoryID  *int64
}

func NewExpense(
	id int64,
	source, description, currency string,
	amount int64,
	date time.Time,
	expenseType ExpenseType,
	categoryID *int64,
) Expense {
	return &expense{
		id:          id,
		source:      source,
		description: description,
		amount:      amount,
		date:        date,
		expenseType: expenseType,
		currency:    currency,
		categoryID:  categoryID,
	}
}

func (e *expense) ID() int64 {
	return e.id
}

func (e *expense) Source() string {
	return e.source
}

func (e *expense) Date() time.Time {
	return e.date
}

func (e *expense) Description() string {
	return e.description
}

func (e *expense) Amount() int64 {
	return e.amount
}

func (e *expense) Type() ExpenseType {
	return e.expenseType
}

func (e *expense) Currency() string {
	return e.currency
}

func (e *expense) CategoryID() *int64 {
	return e.categoryID
}

const ExcludeCategory = "🚫 Exclude"

type Storage interface {
	// Migrations
	ApplyMigrations(*logger.Logger) error

	// Expenses
	GetExpenseByID(int64) (Expense, error)
	UpdateExpense(Expense) (int64, error)
	DeleteExpense(int64) (int64, error)
	InsertExpenses([]Expense) (int64, error)
	GetExpenses() ([]Expense, error)
	GetAllExpenseTypes() ([]Expense, error)
	UpdateExpenses([]Expense) (int64, error)
	GetExpensesFromDateRange(start time.Time, end time.Time) ([]Expense, error)
	GetExpensesWithoutCategory() ([]Expense, error)
	SearchExpenses(keyword string) ([]Expense, error)
	SearchExpensesByDescription(description string) ([]Expense, error)
	GetFirstExpense() (Expense, error)
	GetExpensesByCategory(categoryID int64) ([]Expense, error)

	// Categories
	GetCategories() ([]Category, error)
	GetCategory(categoryID int64) (Category, error)
	DeleteCategory(categoryID int64) (int64, error)
	UpdateCategory(categoryID int64, name, pattern string) error
	CreateCategory(name, pattern string) (int64, error)
	DeleteCategories() (int64, error)
	GetExcludeCategory() (Category, error)

	// Resource managment
	Close() error
}
