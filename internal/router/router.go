package router

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/GustavoCaso/expensetrace/assets"
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
	secureCookie    bool
	html            *htmlRenderer
}

func New(storage storage.Storage, logger *logger.Logger) http.Handler {
	router := newRouter(storage, logger)

	htmlRenderer, err := newHTMLRenderer(assets.HTMLFiles, "base.html", "partials/*.html", "partials/*/*.html")
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	router.html = htmlRenderer

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

	// Create a file server that serves the files from assets/static.

	fileserver := http.FileServerFS(assets.StaticFiles)

	mux.Handle("GET /static/", http.StripPrefix("/static/", fileserver))

	allowEmbedding := os.Getenv("EXPENSETRACE_ALLOW_EMBEDDING") == "true"

	// wrap entire mux with middlewares
	wrappedMux := authMiddleware(router, mux)
	wrappedMux = loggingMiddleware(logger, wrappedMux)

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

// categoryMatcher builds a matcher from the user's current categories. It is
// constructed on demand so it always reflects the latest category patterns.
func (r *router) categoryMatcher(ctx context.Context, userID int64) (*matcher.Matcher, error) {
	categories, err := r.categoryService.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	return matcher.New(categories), nil
}

// renderHTML renders the named template writing the result to w, logging any
// rendering error. render only writes to w on success, so on failure we can
// still send a plain error response.
//
//nolint:unparam // status is always OK today; kept so handlers can return other codes
func (r *router) renderHTML(w http.ResponseWriter, status int, data any, templateName string, files ...string) {
	if err := r.html.render(w, status, data, templateName, files...); err != nil {
		r.logger.Error("Failed to render template", "template", templateName, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
