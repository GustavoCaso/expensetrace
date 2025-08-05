package handler

import (
	"net/http"
	"sort"
	"time"

	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
)

type expensesByYear map[int]map[string][]*expenseDB.Expense

var months = []string{
	time.December.String(),
	time.November.String(),
	time.October.String(),
	time.September.String(),
	time.August.String(),
	time.July.String(),
	time.June.String(),
	time.May.String(),
	time.April.String(),
	time.March.String(),
	time.February.String(),
	time.January.String(),
}

func (h *Handler) expensesHandler(w http.ResponseWriter) {
	expenses, err := expenseDB.GetExpenses(h.db)
	if err != nil {
		data := struct {
			Error error
		}{
			Error: err,
		}
		h.templates.Render(w, "pages/expenses/index.html", data)
	}

	groupedExpenses, years := expensesGroupByYearAndMonth(expenses)
	today := time.Now()

	data := struct {
		Expenses     expensesByYear
		Years        []int
		Months       []string
		Error        error
		CurrentYear  int
		CurrentMonth string
	}{
		Expenses:     groupedExpenses,
		Years:        years,
		Months:       months,
		CurrentYear:  today.Year(),
		CurrentMonth: today.Month().String(),
	}

	h.templates.Render(w, "pages/expenses/index.html", data)
}

func expensesGroupByYearAndMonth(expenses []*expenseDB.Expense) (expensesByYear, []int) {
	groupedExpenses := expensesByYear{}
	years := []int{}

	for _, expense := range expenses {
		expenseYear := expense.Date.Year()
		expenseMonth := expense.Date.Month().String()

		year, okYear := groupedExpenses[expenseYear]

		if okYear {
			month, okMonth := year[expenseMonth]
			if okMonth {
				month = append(month, expense)
			} else {
				month = []*expenseDB.Expense{
					expense,
				}
			}

			year[expenseMonth] = month
		} else {
			years = append(years, expenseYear)
			newYear := map[string][]*expenseDB.Expense{}
			newYear[expenseMonth] = []*expenseDB.Expense{
				expense,
			}
			groupedExpenses[expenseYear] = newYear
		}
	}

	sort.SliceStable(years, func(i, j int) bool {
		return years[i] > years[j]
	})

	return groupedExpenses, years
}
