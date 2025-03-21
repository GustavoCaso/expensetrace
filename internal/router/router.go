package router

import (
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/GustavoCaso/expensetrace/internal/category"
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
