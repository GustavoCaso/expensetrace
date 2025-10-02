package router

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"sync"

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

//go:embed templates/static/*
var static embed.FS
var staticFS, _ = fs.Sub(static, "templates/static")

type router struct {
	matcher   *matcher.Matcher
	storage   storage.Storage
	logger    *logger.Logger
	templates *templates

	reports          map[string]report.Report
	sortedReportKeys []string
	reportsOnce      *sync.Once
	reload           bool
}

//nolint:revive // We return the private router struct to allow testing some internal functions
func New(storage storage.Storage, matcher *matcher.Matcher, logger *logger.Logger) (http.Handler, *router) {
	router := &router{
		reload:      os.Getenv("EXPENSE_LIVERELOAD") == "true",
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

	reports := &reportHandler{
		router,
	}

	categories := &categoryHandler{
		router,
	}

	expenses := &expenseHandler{
		router,
	}

	importHanlder := &importHandler{
		router:       router,
		sessionStore: nil, // Will be initialized in RegisterRoutes
	}

	reports.RegisterRoutes(mux)
	importHanlder.RegisterRoutes(mux)
	expenses.RegisterRoutes(mux)
	categories.RegisterRoutes(mux)

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

func (r *router) resetCache() {
	r.reportsOnce = &sync.Once{}
}

func (r *router) parseTemplates() error {
	t := &templates{
		t:      map[string]*template.Template{},
		logger: r.logger,
	}

	var fs fs.FS
	var err error
	if r.reload {
		fs, err = localFSDirectory(r.logger)
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

	r.templates = t
	return nil
}
