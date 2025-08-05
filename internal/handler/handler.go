package handler

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

type Handler struct {
	HTTPHandler      http.Handler
	reload           bool
	matcher          *category.Matcher
	db               *sql.DB
	templates        templates
	reports          map[string]report.Report
	sortedReportKeys []string
	reportsOnce      *sync.Once
	logger           *logger.Logger
}

//nolint:revive // We return the private router struct to allow testing some internal functions
func New(db *sql.DB, matcher *category.Matcher, logger *logger.Logger) *Handler {
	allowEmbedding := os.Getenv("EXPENSETRACE_ALLOW_EMBEDDING") == "true"

	handler := &Handler{
		reload:      os.Getenv("LIVERELOAD") == "true",
		matcher:     matcher,
		db:          db,
		reportsOnce: &sync.Once{},
		logger:      logger,
	}

	mux := &http.ServeMux{}

	parseError := handler.parseTemplates()

	if parseError != nil {
		logger.Fatal("error parsing templates", "error", parseError.Error())
	}

	// Routes
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		handler.reportsOnce.Do(func() {
			err := handler.generateReports()

			if err != nil {
				// If we fail to generate reports servers do not start
				// TODO: fix
				logger.Fatal("Failed to generate reports", "error", err)
			}

			reportKeys := slices.Collect(maps.Keys(handler.reports))

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

			handler.sortedReportKeys = reportKeys
		})
		handler.homeHandler(w, r)
	})

	mux.HandleFunc("DELETE /expense/{id}", func(_ http.ResponseWriter, _ *http.Request) {
		// TODO
	})

	mux.HandleFunc("GET /expenses", func(w http.ResponseWriter, _ *http.Request) {
		handler.expensesHandler(w)
	})

	mux.HandleFunc("GET /import", func(w http.ResponseWriter, _ *http.Request) {
		handler.templates.Render(w, "pages/import/index.html", nil)
	})

	mux.HandleFunc("POST /import", func(w http.ResponseWriter, r *http.Request) {
		handler.importHandler(w, r)
	})

	mux.HandleFunc("GET /categories", func(w http.ResponseWriter, _ *http.Request) {
		handler.categoriesHandler(w)
	})

	mux.HandleFunc("GET /category/new", func(w http.ResponseWriter, _ *http.Request) {
		handler.templates.Render(w, "pages/categories/new.html", nil)
	})

	mux.HandleFunc("GET /category/uncategorized", func(w http.ResponseWriter, _ *http.Request) {
		handler.uncategorizedHandler(w)
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
			handler.templates.Render(w, "partials/categories/card.html", data)
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

		handler.updateCategoryHandler(categoryID, name, pattern, categoryType, w)
	})

	mux.HandleFunc("POST /category/check", func(w http.ResponseWriter, r *http.Request) {
		handler.createCategoryHandler(false, w, r)
	})

	mux.HandleFunc("POST /category", func(w http.ResponseWriter, r *http.Request) {
		handler.createCategoryHandler(true, w, r)
	})

	mux.HandleFunc("POST /category/uncategorized/update", func(w http.ResponseWriter, r *http.Request) {
		handler.updateUncategorizedHandler(w, r)
	})

	mux.HandleFunc("POST /search", func(w http.ResponseWriter, r *http.Request) {
		handler.searchHandler(w, r)
	})

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// wrap entire mux with middlewares
	wrappedHandler := loggingMiddleware(logger, mux)
	wrappedHandler = liveReloadMiddleware(handler, wrappedHandler)
	if !allowEmbedding {
		wrappedHandler = xFrameDenyHeaderMiddleware(wrappedHandler)
	}

	handler.HTTPHandler = wrappedHandler

	return handler
}

func (h *Handler) generateReports() error {
	now := time.Now()
	month := now.Month()
	year := now.Year()
	skipYear := false
	ex, err := expenseDB.GetFirstExpense(h.db)
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

		expenses, expenseErr := expenseDB.GetExpensesFromDateRange(h.db, firstDay, lastDay)

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

	h.reports = reports

	return nil
}

func (h *Handler) resetCache() {
	h.reportsOnce = &sync.Once{}
}
