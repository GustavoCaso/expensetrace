package router

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

type router struct {
	reload bool
	mux    *http.ServeMux
}

func New(db *sql.DB, matcher *category.Matcher) *http.ServeMux {
	reload := os.Getenv("LIVERELOAD") == "true"

	r := router{
		reload: reload,
		mux:    &http.ServeMux{},
	}

	r.parseTemplates()

	// Routes
	r.mux.Handle("GET /", r.liveReloadTemplatesMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		homeHandler(db, w, r)
	})))

	r.mux.Handle("GET /expenses", r.liveReloadTemplatesMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		expensesHandler(db, w)
	})))

	r.mux.Handle("GET /import", r.liveReloadTemplatesMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		err := importTempl.Execute(w, nil)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})))

	r.mux.Handle("GET /categories", r.liveReloadTemplatesMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		categoriesHandler(db, w)
	})))

	r.mux.Handle("GET /uncategorized", r.liveReloadTemplatesMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		uncategorizedHandler(db, matcher, w)
	})))

	r.mux.Handle("POST /category", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		updateCategoryHandler(db, matcher, w, r)
	}))

	r.mux.Handle("POST /search", r.liveReloadTemplatesMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		searchHandler(db, w, r)
	})))

	r.mux.Handle("POST /import", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		importHandler(db, matcher, w, r)
	}))

	return r.mux
}

type homeData struct {
	Report report.Report
	Error  error
}

func homeHandler(db *sql.DB, w http.ResponseWriter, _ *http.Request) {
	// Fetch expenses from last month
	now := time.Now()

	firstDay, lastDay := util.GetMonthDates(int(now.Month()-1), now.Year())

	var data homeData
	expenses, err := expenseDB.GetExpensesFromDateRange(db, firstDay, lastDay)

	if err != nil {
		data = homeData{
			Error: err,
		}
	} else {
		result := report.Generate(firstDay, lastDay, expenses, "monthly")

		data = homeData{
			Report: result,
			Error:  nil,
		}
	}

	err = indexTempl.Execute(w, data)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
