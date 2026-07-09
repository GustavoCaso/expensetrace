package domain

import (
	"encoding/json"
	"time"
)

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

type ExpenseView struct {
	Expense
	Cat Category
}

func (e *ExpenseView) Category() Category { return e.Cat }

func (e *ExpenseView) CategoryID() int64 {
	if e.Expense.CategoryID() != nil {
		return *e.Expense.CategoryID()
	}
	return 0
}

type ExpensesByYear map[int]map[string][]*ExpenseView

type ExpensesViewData struct {
	ViewBase
	Expenses     ExpensesByYear
	Years        []int
	Months       []string
	CurrentYear  int
	CurrentMonth string
	Filter       *ExpenseFilter
	Sort         *SortOptions
}

type ExpenseViewData struct {
	ViewBase
	Expense    *ExpenseView
	Categories []Category
	FormErrors map[string]string
	Action     string
	RedirectTo string
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

func (e *expense) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"id":           e.id,
		"source":       e.source,
		"date":         e.date,
		"description":  e.description,
		"amount":       e.amount,
		"expense_type": e.expenseType,
		"currency":     e.currency,
		"category_id":  e.categoryID,
	})
}
