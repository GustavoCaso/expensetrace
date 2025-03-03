package router

import (
	"fmt"
	"log"
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

func (router *router) expensesHandler(w http.ResponseWriter) {
	expenses, err := expenseDB.GetExpenses(router.db)
	if err != nil {
		data := struct {
			Error error
		}{
			Error: err,
		}
		err = router.templates.Render(w, "pages/expenses/index.html", data)
		if err != nil {
			log.Print(err.Error())
			errorMessage := fmt.Sprintf("Internal Server Error: %v", err.Error())
			w.Write([]byte(errorMessage))
			return
		}
	}

	groupedExpenses, years := expensesGroupByYearAndMonth(expenses)

	data := struct {
		Expenses expensesByYear
		Years    []int
		Months   []string
		Error    error
	}{
		Expenses: groupedExpenses,
		Years:    years,
		Months:   months,
	}

	err = router.templates.Render(w, "pages/expenses/index.html", data)
	if err != nil {
		log.Print(err.Error())
		errorMessage := fmt.Sprintf("Internal Server Error: %v", err.Error())
		w.Write([]byte(errorMessage))
	}
}

func expensesGroupByYearAndMonth(expenses []*expenseDB.Expense) (expensesByYear, []int) {
	groupedExpenses := expensesByYear{}
	years := []int{}

	for _, expense := range expenses {
		expenseYear := expense.Date.Year()
		expenseMonth := expense.Date.Month().String()

		year, ok := groupedExpenses[expenseYear]

		if ok {
			month, ok := year[expenseMonth]
			if ok {
				month = append(month, expense)
			} else {
				month = []*expenseDB.Expense{
					expense,
				}
			}

			year[expenseMonth] = month
		} else {
			years = append(years, expenseYear)
			year := map[string][]*expenseDB.Expense{}
			year[expenseMonth] = []*expenseDB.Expense{
				expense,
			}
			groupedExpenses[expenseYear] = year
		}
	}

	sort.SliceStable(years, func(i, j int) bool {
		return years[i] > years[j]
	})

	return groupedExpenses, years
}
