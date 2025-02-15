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
	reload bool
	mux    *http.ServeMux
}

func New(db *sql.DB, matcher *category.Matcher) *http.ServeMux {
	r := router{
		reload: os.Getenv("LIVERELOAD") == "true",
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

type link struct {
	Name string
	URL  string
}

type homeData struct {
	Report report.Report
	Links  []link
	Error  error
}

func homeHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
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
	expenses, err := expenseDB.GetExpensesFromDateRange(db, firstDay, lastDay)

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
