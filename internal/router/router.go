package router

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

//go:embed templates/static/*
var static embed.FS
var staticFS, _ = fs.Sub(static, "templates/static")

type router struct {
	reload           bool
	matcher          *category.Matcher
	db               *sql.DB
	templates        templates
	reports          map[string]report.Report
	sortedReportKeys []string
	reportsOnce      *sync.Once
}

//nolint:revive // We return the private router struct to allow testing some internal functions
func New(db *sql.DB, matcher *category.Matcher) (http.Handler, *router) {
	router := &router{
		reload:      os.Getenv("LIVERELOAD") == "true",
		matcher:     matcher,
		db:          db,
		reportsOnce: &sync.Once{},
	}

	mux := &http.ServeMux{}

	parseError := router.parseTemplates()

	if parseError != nil {
		panic(parseError)
	}

	// Routes
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		router.reportsOnce.Do(func() {
			err := router.generateReports()

			if err != nil {
				// If we fail to generate reports servers do not start
				// TODO: fix
				log.Fatalf("generateReports fail %v", err)
			}

			reportKeys := slices.Collect(maps.Keys(router.reports))

			sort.SliceStable(reportKeys, func(i, j int) bool {
				s1 := strings.Split(reportKeys[i], "-")
				s2 := strings.Split(reportKeys[j], "-")
				year1, _ := strconv.Atoi(s1[0])
				month1, _ := strconv.Atoi(s1[1])

				year2, _ := strconv.Atoi(s2[0])
				month2, _ := strconv.Atoi(s2[1])

				if year1 == year2 {
					return time.Month(month1) > time.Month(month2)
				}

				return year1 > year2
			})

			router.sortedReportKeys = reportKeys
		})
		router.homeHandler(w, r)
	})

	mux.HandleFunc("DELETE /expense/{id}", func(_ http.ResponseWriter, _ *http.Request) {
		// TODO
	})

	mux.HandleFunc("GET /expenses", func(w http.ResponseWriter, _ *http.Request) {
		router.expensesHandler(w)
	})

	mux.HandleFunc("GET /import", func(w http.ResponseWriter, _ *http.Request) {
		router.templates.Render(w, "pages/import/index.html", nil)
	})

	mux.HandleFunc("POST /import", func(w http.ResponseWriter, r *http.Request) {
		router.importHandler(w, r)
	})

	mux.HandleFunc("GET /categories", func(w http.ResponseWriter, _ *http.Request) {
		router.categoriesHandler(w)
	})

	mux.HandleFunc("GET /category/new", func(w http.ResponseWriter, _ *http.Request) {
		router.templates.Render(w, "pages/categories/new.html", nil)
	})

	mux.HandleFunc("GET /category/uncategorized", func(w http.ResponseWriter, _ *http.Request) {
		router.uncategorizedHandler(w)
	})

	mux.HandleFunc("PUT /category/{id}", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Println("error r.ParseForm() ", err.Error())

			data := struct {
				Error error
			}{
				Error: err,
			}
			router.templates.Render(w, "partials/categories/card.html", data)
			return
		}

		categoryID := r.PathValue("id")
		name := r.FormValue("name")
		pattern := r.FormValue("pattern")
		categoryTypeStr := r.FormValue("type")
		categoryType := expenseDB.ExpenseCategoryType // Default to expense
		if categoryTypeStr == "1" {
			categoryType = expenseDB.IncomeCategoryType
		}

		router.updateCategoryHandler(categoryID, name, pattern, categoryType, w)
	})

	mux.HandleFunc("POST /category/check", func(w http.ResponseWriter, r *http.Request) {
		router.createCategoryHandler(false, w, r)
	})

	mux.HandleFunc("POST /category", func(w http.ResponseWriter, r *http.Request) {
		router.createCategoryHandler(true, w, r)
	})

	mux.HandleFunc("POST /category/uncategorized/update", func(w http.ResponseWriter, r *http.Request) {
		router.updateUncategorizedHandler(w, r)
	})

	mux.HandleFunc("POST /search", func(w http.ResponseWriter, r *http.Request) {
		router.searchHandler(w, r)
	})

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// wrap entire mux with live reload middleware
	liveReloadMux := newLiveReloadMiddleware(router, mux)

	return liveReloadMux, router
}

func (router *router) generateReports() error {
	now := time.Now()
	month := now.Month()
	year := now.Year()
	skipYear := false
	ex, err := expenseDB.GetFirstExpense(router.db)
	if err != nil {
		return err
	}

	lastMonth := ex.Date.Month()
	lastYear := ex.Date.Year()

	reports := map[string]report.Report{}

	for year >= lastYear {
		if month == time.January {
			skipYear = true
		}

		firstDay, lastDay := util.GetMonthDates(int(month), year)

		expenses, expenseErr := expenseDB.GetExpensesFromDateRange(router.db, firstDay, lastDay)

		if expenseErr != nil {
			return expenseErr
		}

		result := report.Generate(firstDay, lastDay, expenses, "monthly")

		reports[fmt.Sprintf("%d-%d", year, month)] = result

		if skipYear {
			year--
			month = time.December
			skipYear = false
			continue
		}

		if year == lastYear && month == lastMonth {
			break
		}

		month--
	}

	router.reports = reports

	return nil
}

func (router *router) resetCache() {
	router.reportsOnce = &sync.Once{}
}
