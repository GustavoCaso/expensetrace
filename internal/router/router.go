package router

import (
	"html/template"
	"io/fs"
	"net/http"
	"os"

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

type router struct {
	matcher   *matcher.Matcher
	storage   storage.Storage
	logger    *logger.Logger
	templates *templates

	reload bool
}

//nolint:revive // We return the private router struct to allow testing some internal functions
func New(storage storage.Storage, logger *logger.Logger) (http.Handler, *router) {
	router := &router{
		reload:  os.Getenv("EXPENSE_LIVERELOAD") == "true",
		storage: storage,
		logger:  logger,
	}

	staticFS, staticFSError := router.parserStaticFiles()

	if staticFSError != nil {
		logger.Fatal("error parsing static files", "error", staticFSError.Error())
	}

	parseError := router.parseTemplates()

	if parseError != nil {
		logger.Fatal("error parsing templates", "error", parseError.Error())
	}

	reports := newReportsHandlder(router)

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

	auth := &authHandler{
		router,
	}

	profile := &profileHandler{
		router,
	}

	mux := &http.ServeMux{}

	// Register auth routes first (these will be excluded from auth middleware)
	auth.RegisterRoutes(mux)

	reports.RegisterRoutes(mux)
	importHanlder.RegisterRoutes(mux)
	expenses.RegisterRoutes(mux)
	categories.RegisterRoutes(mux)
	profile.RegisterRoutes(mux)

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	allowEmbedding := os.Getenv("EXPENSETRACE_ALLOW_EMBEDDING") == "true"

	// wrap entire mux with middlewares
	wrappedMux := authMiddleware(router, storage, logger, mux)
	wrappedMux = loggingMiddleware(logger, wrappedMux)

	if router.reload {
		wrappedMux = liveReloadMiddleware(router, wrappedMux)
	}

	if !allowEmbedding {
		wrappedMux = xFrameDenyHeaderMiddleware(wrappedMux)
	}

	return wrappedMux, router
}

func (r *router) parseTemplates() error {
	t := &templates{
		t:      map[string]*template.Template{},
		logger: r.logger,
	}

	var fs fs.FS
	var err error
	if r.reload {
		fs, err = localFSDirectory(r.logger, "../templates")
	} else {
		fs, err = embeddedFS("templates")
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

func (r *router) parserStaticFiles() (fs.FS, error) {
	var fs fs.FS
	var err error
	if r.reload {
		fs, err = localFSDirectory(r.logger, "../templates/static")
		if err != nil {
			r.logger.Warn("Failed to load local static files, falling back to embedded", "error", err.Error())
			fs, err = embeddedFS("templates/static")
		}
	} else {
		fs, err = embeddedFS("templates/static")
	}

	return fs, err
}
