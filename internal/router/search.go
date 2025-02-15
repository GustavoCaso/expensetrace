package router

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sort"

	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/report"
)

func searchHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	query := r.FormValue("q")

	if query == "" {
		fmt.Fprint(w, "You must provide a search criteria")
		return
	}

	expenses, err := expenseDB.SearchExpenses(db, query)
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

	err = searchResultsTempl.ExecuteTemplate(w, "searchResults.html", data)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

}
