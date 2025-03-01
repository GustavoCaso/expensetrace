package router

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
)

func (router *router) expensesHandler(w http.ResponseWriter) {
	expenses, err := expenseDB.GetExpenses(router.db)
	if err != nil {
		data := struct {
			Error error
		}{
			Error: err,
		}
		err = router.templates.Render(w, "pages/expenses.html", data)
		if err != nil {
			log.Print(err.Error())
			errorMessage := fmt.Sprintf("Internal Server Error: %v", err.Error())
			w.Write([]byte(errorMessage))
			return
		}
	}

	sort.Slice(expenses, func(i, j int) bool {
		return expenses[i].Date.Unix() > expenses[j].Date.Unix()
	})

	data := struct {
		Expenses []*expenseDB.Expense
		Error    error
	}{
		Expenses: expenses,
	}

	err = router.templates.Render(w, "pages/expenses.html", data)
	if err != nil {
		log.Print(err.Error())
		errorMessage := fmt.Sprintf("Internal Server Error: %v", err.Error())
		w.Write([]byte(errorMessage))
	}
}
