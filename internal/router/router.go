package router

import (
	"embed"
	"fmt"
	"html/template"
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

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

//go:embed templates/static/*
var static embed.FS
var staticFS, _ = fs.Sub(static, "templates/static")

type router struct {
	reload           bool
	matcher          *matcher.Matcher
	storage          storage.Storage
	templates        *templates
	reports          map[string]report.Report
	sortedReportKeys []string
	reportsOnce      *sync.Once
	logger           *logger.Logger
}

//nolint:revive // We return the private router struct to allow testing some internal functions
func New(storage storage.Storage, matcher *matcher.Matcher, logger *logger.Logger) (http.Handler, *router) {
	router := &router{
		reload:      os.Getenv("LIVERELOAD") == "true",
		matcher:     matcher,
		storage:     storage,
		reportsOnce: &sync.Once{},
		logger:      logger,
	}

	mux := &http.ServeMux{}

	parseError := router.parseTemplates()

	if parseError != nil {
		logger.Fatal("error parsing templates", "error", parseError.Error())
	}

	// Routes
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		router.reportsOnce.Do(func() {
			err := router.generateReports()

			if err != nil {
				logger.Warn("Failed to generate reports", "error", err)
				return
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

	mux.HandleFunc("GET /expense/{id}", func(w http.ResponseWriter, r *http.Request) {
		router.expenseHandler(w, r)
	})

	mux.HandleFunc("PUT /expense/{id}", func(w http.ResponseWriter, r *http.Request) {
		router.updateExpenseHandler(w, r)
	})

	mux.HandleFunc("DELETE /expense/{id}", func(w http.ResponseWriter, r *http.Request) {
		router.deleteExpenseHandler(w, r)
	})

	mux.HandleFunc("GET /expenses", func(w http.ResponseWriter, _ *http.Request) {
		router.expensesHandler(w)
	})

	mux.HandleFunc("POST /expense/search", func(w http.ResponseWriter, r *http.Request) {
		router.expenseSearchHandler(w, r)
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
			logger.Error("Failed to parse form", "error", err)

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
		// Category type is no longer needed - we only handle expenses
		router.updateCategoryHandler(categoryID, name, pattern, w)
	})

	mux.HandleFunc("POST /category/check", func(w http.ResponseWriter, r *http.Request) {
		router.createCategoryHandler(false, w, r)
	})

	mux.HandleFunc("POST /category", func(w http.ResponseWriter, r *http.Request) {
		router.createCategoryHandler(true, w, r)
	})

	mux.HandleFunc("POST /category/reset", func(w http.ResponseWriter, _ *http.Request) {
		router.resetCategoryHandler(w)
	})

	mux.HandleFunc("POST /category/uncategorized/update", func(w http.ResponseWriter, r *http.Request) {
		router.updateUncategorizedHandler(w, r)
	})

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	allowEmbedding := os.Getenv("EXPENSETRACE_ALLOW_EMBEDDING") == "true"

	// wrap entire mux with middlewares
	wrappedMux := loggingMiddleware(logger, mux)

	if router.reload {
		wrappedMux = liveReloadMiddleware(router, wrappedMux)
	}

	if !allowEmbedding {
		wrappedMux = xFrameDenyHeaderMiddleware(wrappedMux)
	}

	return wrappedMux, router
}

func (router *router) generateReports() error {
	now := time.Now()
	month := now.Month()
	year := now.Year()
	skipYear := false
	ex, err := router.storage.GetFirstExpense()
	if err != nil {
		return err
	}

	lastMonth := ex.Date().Month()
	lastYear := ex.Date().Year()

	reports := map[string]report.Report{}

	for year >= lastYear {
		if month == time.January {
			skipYear = true
		}

		firstDay, lastDay := util.GetMonthDates(int(month), year)

		expenses, expenseErr := router.storage.GetExpensesFromDateRange(firstDay, lastDay)

		if expenseErr != nil {
			return expenseErr
		}

		result, reportErr := report.Generate(firstDay, lastDay, router.storage, expenses, "monthly")

		if reportErr != nil {
			return reportErr
		}

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

func (router *router) parseTemplates() error {
	t := &templates{
		t:      map[string]*template.Template{},
		logger: router.logger,
	}

	var fs fs.FS
	var err error
	if router.reload {
		fs, err = localFSDirectory(router.logger)
	} else {
		fs, err = embeddedFS()
	}

	if err != nil {
		return err
	}

	err = t.parseTemplates(fs)

	if err != nil {
		return err
	}

	router.templates = t
	return nil
}
