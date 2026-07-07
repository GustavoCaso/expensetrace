package domain

import (
	"github.com/GustavoCaso/expensetrace/internal/filter"
	pkgStorage "github.com/GustavoCaso/expensetrace/internal/storage"
)

type ExpenseView struct {
	pkgStorage.Expense
	Cat pkgStorage.Category
}

func (e *ExpenseView) Category() pkgStorage.Category { return e.Cat }

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
	Filter       *filter.ExpenseFilter
	Sort         *filter.SortOptions
}

type ExpenseViewData struct {
	ViewBase
	Expense    *ExpenseView
	Categories []pkgStorage.Category
	FormErrors map[string]string
	Action     string
	RedirectTo string
}
