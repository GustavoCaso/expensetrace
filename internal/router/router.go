package router

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
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

//go:embed templates/static/*
var static embed.FS
var staticFS, _ = fs.Sub(static, "templates/static")

type router struct {
	reload    bool
	mux       *http.ServeMux
	matcher   *category.Matcher
	db        *sql.DB
	templates templates
}

func New(db *sql.DB, matcher *category.Matcher) http.Handler {
	router := &router{
		reload:  os.Getenv("LIVERELOAD") == "true",
		matcher: matcher,
		db:      db,
	}

	mux := &http.ServeMux{}

	router.parseTemplates()

	// Routes
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		router.homeHandler(w, r)
	})

	mux.HandleFunc("GET /expenses", func(w http.ResponseWriter, _ *http.Request) {
		router.expensesHandler(w)
	})

	mux.HandleFunc("GET /import", func(w http.ResponseWriter, _ *http.Request) {
		err := router.templates.Render(w, "pages/import/index.html", nil)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("GET /categories", func(w http.ResponseWriter, _ *http.Request) {
		router.categoriesHandler(w)
	})

	mux.HandleFunc("GET /category/uncategorized", func(w http.ResponseWriter, _ *http.Request) {
		router.uncategorizedHandler(w)
	})

	mux.HandleFunc("GET /category/new", func(w http.ResponseWriter, _ *http.Request) {
		err := router.templates.Render(w, "pages/categories/new.html", nil)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("POST /category/check", func(w http.ResponseWriter, r *http.Request) {
		router.createCategoryHandler(false, w, r)
	})

	mux.HandleFunc("POST /category", func(w http.ResponseWriter, r *http.Request) {
		router.createCategoryHandler(true, w, r)
	})

	mux.HandleFunc("POST /category/uncategorized/update", func(w http.ResponseWriter, r *http.Request) {
		router.updateCategoryHandler(w, r)
	})

	mux.HandleFunc("POST /search", func(w http.ResponseWriter, r *http.Request) {
		router.searchHandler(w, r)
	})

	mux.HandleFunc("POST /import", func(w http.ResponseWriter, r *http.Request) {
		router.importHandler(w, r)
	})

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	//wrap entire mux with live reload middleware
	wrappedMux := newLiveReloadMiddleware(router, mux)

	return wrappedMux
}

type link struct {
	Name     string
	URL      string
	Income   int64
	Spending int64
	Savings  int64
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
		links, err = generateLinks(router.db, time.Month(month), year)
	}

	var data homeData
	if err != nil {
		data = homeData{
			Error: err,
		}
	} else {
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
	}

	if useReportTemplate {
		err = router.templates.Render(w, "partials/reports/report.html", data)
	} else {
		err = router.templates.Render(w, "pages/reports/index.html", data)
	}

	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func generateLinks(db *sql.DB, month time.Month, year int) ([]link, error) {
	links := []link{}
	skipYear := false
	timeMonth := time.Month(month)
	for year > 2022 {
		if timeMonth == time.January {
			skipYear = true
		}

		firstDay, lastDay := util.GetMonthDates(int(timeMonth), year)

		expenses, err := expenseDB.GetExpensesFromDateRange(db, firstDay, lastDay)

		if err != nil {
			return []link{}, err
		}

		result := report.Generate(firstDay, lastDay, expenses, "monthly")

		links = append(links, link{
			Name:     fmt.Sprintf("%s %d", timeMonth, year),
			URL:      fmt.Sprintf("/?month=%d&year=%d", int(timeMonth), year),
			Income:   result.Income,
			Spending: result.Spending,
			Savings:  result.Savings,
		})

		if skipYear {
			year = year - 1
			timeMonth = time.December
			skipYear = false
			continue
		}

		timeMonth = timeMonth - 1
	}

	return links, nil
}
