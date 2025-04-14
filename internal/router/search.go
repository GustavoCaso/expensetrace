package router

import (
	"fmt"
	"net/http"
	"sort"

	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/report"
)

func (router *router) searchHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()

	if err != nil {
		fmt.Fprint(w, "error r.ParseForm() ", err.Error())
		return
	}

	query := r.FormValue("q")

	if query == "" {
		fmt.Fprint(w, "You must provide a search criteria")
		return
	}

	expenses, err := expenseDB.SearchExpenses(router.db, query)
	if err != nil {
		panic(err)
	}

	sort.Slice(expenses, func(i, j int) bool {
		return expenses[i].Date.Unix() < expenses[j].Date.Unix()
	})

	categories, _, _, _ := report.Categories(expenses)

	data := struct {
		Categories map[string]report.Category
		Query      string
	}{
		Categories: categories,
		Query:      query,
	}

	router.templates.Render(w, "partials/search/results.html", data)
}
