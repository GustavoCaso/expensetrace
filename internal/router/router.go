package router

import (
	"context"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/service/auth"
	"github.com/GustavoCaso/expensetrace/internal/service/category"
	"github.com/GustavoCaso/expensetrace/internal/service/expense"
	"github.com/GustavoCaso/expensetrace/internal/service/importsvc"
	"github.com/GustavoCaso/expensetrace/internal/service/profile"
	"github.com/GustavoCaso/expensetrace/internal/service/report"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

// importSessionTTL is how long an in-progress import session is kept alive
// before it expires.
const importSessionTTL = 30 * time.Minute

type router struct {
	logger          *logger.Logger
	categoryService *category.Service
	expenseService  *expense.Service
	reportService   *report.Service
	importService   *importsvc.Service
	authService     *auth.Service
	profileService  *profile.Service
	templates       *templates
	secureCookie    bool
	reload          bool
}

func New(storage storage.Storage, logger *logger.Logger) http.Handler {
	router := newRouter(storage, logger)

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
		router,
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
	wrappedMux := authMiddleware(router, mux)
	wrappedMux = loggingMiddleware(logger, wrappedMux)

	if router.reload {
		wrappedMux = liveReloadMiddleware(router, wrappedMux)
	}

	if !allowEmbedding {
		wrappedMux = xFrameDenyHeaderMiddleware(wrappedMux)
	}

	wrappedMux = csrfProtectionMiddleware(logger, wrappedMux)

	return wrappedMux
}

// newRouter builds the router with its services. Split from New so tests can
// exercise internal functions directly.
func newRouter(storage storage.Storage, logger *logger.Logger) *router {
	router := &router{
		reload:          os.Getenv("EXPENSE_LIVERELOAD") == "true",
		secureCookie:    os.Getenv("EXPENSETRACE_SECURE_COOKIES") == "true",
		logger:          logger,
		categoryService: category.New(storage, logger),
		expenseService:  expense.New(storage, logger),
		reportService:   report.New(storage, logger),
		importService:   importsvc.New(storage, logger, importSessionTTL),
		authService:     auth.New(storage, logger),
		profileService:  profile.New(storage, logger),
	}

	return router
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

// categoryMatcher builds a matcher from the user's current categories. It is
// constructed on demand so it always reflects the latest category patterns.
func (r *router) categoryMatcher(ctx context.Context, userID int64) (*matcher.Matcher, error) {
	categories, err := r.categoryService.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	return matcher.New(categories), nil
}
