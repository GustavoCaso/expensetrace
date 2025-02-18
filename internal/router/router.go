package router

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

type router struct {
	reload  bool
	mux     *http.ServeMux
	matcher *category.Matcher
	db      *sql.DB
}

func New(db *sql.DB, matcher *category.Matcher) http.Handler {
	router := &router{
		reload:  os.Getenv("LIVERELOAD") == "true",
		mux:     &http.ServeMux{},
		matcher: matcher,
		db:      db,
	}

	router.parseTemplates()

	// Routes
	router.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		router.homeHandler(w, r)
	})

	router.mux.HandleFunc("GET /expenses", func(w http.ResponseWriter, _ *http.Request) {
		router.expensesHandler(w)
	})

	router.mux.HandleFunc("GET /import", func(w http.ResponseWriter, _ *http.Request) {
		err := importTempl.Execute(w, nil)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	router.mux.HandleFunc("GET /categories", func(w http.ResponseWriter, _ *http.Request) {
		router.categoriesHandler(w)
	})

	router.mux.HandleFunc("GET /category/uncategorized", func(w http.ResponseWriter, _ *http.Request) {
		router.uncategorizedHandler(w)
	})

	router.mux.HandleFunc("GET /category/new", func(w http.ResponseWriter, _ *http.Request) {
		err := newCategoriesTempl.Execute(w, nil)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	router.mux.HandleFunc("POST /category/check", func(w http.ResponseWriter, r *http.Request) {
		router.createCategoryHandler(false, w, r)
	})

	router.mux.HandleFunc("POST /category", func(w http.ResponseWriter, r *http.Request) {
		router.createCategoryHandler(true, w, r)
	})

	router.mux.HandleFunc("POST /category/uncategorized/update", func(w http.ResponseWriter, r *http.Request) {
		router.updateCategoryHandler(w, r)
	})

	router.mux.HandleFunc("POST /search", func(w http.ResponseWriter, r *http.Request) {
		router.searchHandler(w, r)
	})

	router.mux.HandleFunc("POST /import", func(w http.ResponseWriter, r *http.Request) {
		router.importHandler(w, r)
	})

	//wrap entire mux with live reload middleware
	wrappedMux := newLiveReloadMiddleware(router, router.mux)

	return wrappedMux
}

type link struct {
	Name string
	URL  string
}

type homeData struct {
	Report report.Report
	Links  []link
	Error  error
}

func (router *router) homeHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	var month int
	var year int
	var err error
	useReportTemplate := false

	monthQuery := r.URL.Query().Get("month")
	if monthQuery != "" {
		if month, err = strconv.Atoi(monthQuery); err != nil {
			fmt.Println("error strconv.Atoi ", err.Error())
			month = int(now.Month() - 1)
		}
		useReportTemplate = true
	} else {
		month = int(now.Month() - 1)
	}
	yearQuery := r.URL.Query().Get("year")
	if yearQuery != "" {
		if year, err = strconv.Atoi(yearQuery); err != nil {
			fmt.Println("error strconv.Atoi ", err.Error())
			year = now.Year()
		}
		useReportTemplate = true
	} else {
		year = now.Year()
	}

	firstDay, lastDay := util.GetMonthDates(month, year)
	var links []link
	if !useReportTemplate {
		links = generateLinks(time.Month(month), year)
	}

	var data homeData
	expenses, err := expenseDB.GetExpensesFromDateRange(router.db, firstDay, lastDay)

	if err != nil {
		data = homeData{
			Error: err,
		}
	} else {
		result := report.Generate(firstDay, lastDay, expenses, "monthly")

		data = homeData{
			Report: result,
			Links:  links,
			Error:  nil,
		}
	}

	if useReportTemplate {
		err = reportTempl.ExecuteTemplate(w, "report.html", data)
	} else {
		err = indexTempl.Execute(w, data)
	}

	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func generateLinks(month time.Month, year int) []link {
	links := []link{}
	skipYear := false
	timeMonth := time.Month(month)
	for year > 2021 {
		if timeMonth == time.January {
			skipYear = true
		}

		links = append(links, link{
			Name: fmt.Sprintf("%s %d", timeMonth, year),
			URL:  fmt.Sprintf("/?month=%d&year=%d", int(timeMonth), year),
		})

		if skipYear {
			year = year - 1
			timeMonth = time.December
			skipYear = false
			continue
		}

		timeMonth = timeMonth - 1
	}

	return links
}
