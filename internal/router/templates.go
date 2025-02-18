package router

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/GustavoCaso/expensetrace/internal/util"
)

// content holds our static content.
//
//go:embed templates
var templatesFS embed.FS

// Templates
var baseTempl *template.Template
var indexTempl *template.Template
var reportTempl *template.Template
var importTempl *template.Template
var searchResultsTempl *template.Template
var expensesTempl *template.Template

var categoriesTempl *template.Template
var uncategoriesTempl *template.Template
var newCategoriesTempl *template.Template
var newCategoryResult *template.Template

var templateFuncs = template.FuncMap{
	"formatMoney": util.FormatMoney,
}

func parseFSTemplates() {
	baseTempl = template.Must(template.New("base").Funcs(templateFuncs).ParseFS(templatesFS, []string{
		"templates/home.html",
		"templates/partials/nav.html",
		"templates/partials/search/form.html",
	}...))

	indexTempl = template.Must(template.Must(baseTempl.Clone()).ParseFS(templatesFS, []string{
		"templates/pages/index.html",
	}...))

	importTempl = template.Must(template.Must(baseTempl.Clone()).ParseFS(templatesFS, []string{
		"templates/pages/import.html",
	}...))

	expensesTempl = template.Must(template.Must(baseTempl.Clone()).ParseFS(templatesFS, []string{
		"templates/pages/expenses.html",
	}...))

	categoriesTempl = template.Must(template.Must(baseTempl.Clone()).ParseFS(templatesFS, []string{
		"templates/pages/categories.html",
	}...))

	newCategoriesTempl = template.Must(template.Must(baseTempl.Clone()).ParseFS(templatesFS, []string{
		"templates/pages/categories/new.html",
	}...))

	newCategoryResult = template.Must(template.New("").Funcs(templateFuncs).ParseFS(templatesFS, []string{
		"templates/partials/categories/new_result.html",
	}...))

	reportTempl = template.Must(template.New("").Funcs(templateFuncs).ParseFS(templatesFS,
		"templates/partials/reports/report.html",
	))

	searchResultsTempl = template.Must(template.New("").Funcs(templateFuncs).ParseFS(templatesFS,
		"templates/partials/search/results.html"),
	)

	uncategoriesTempl = template.Must(template.New("").Funcs(templateFuncs).ParseFS(templatesFS,
		"templates/partials/categories/uncategorized.html"),
	)
}

func parseLocalTemplates() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Printf("enable to get current directory %s. defaulting to embedded templates\n", filename)
		parseFSTemplates()
	}

	baseTempl = template.Must(template.New("base").Funcs(templateFuncs).ParseFiles([]string{
		filepath.Join(filename, "../templates/home.html"),
		filepath.Join(filename, "../templates/partials/nav.html"),
		filepath.Join(filename, "../templates/partials/search/form.html"),
	}...))

	indexTempl = template.Must(template.Must(baseTempl.Clone()).ParseFiles([]string{
		filepath.Join(filename, "../templates/pages/index.html"),
	}...))

	importTempl = template.Must(template.Must(baseTempl.Clone()).ParseFiles([]string{
		filepath.Join(filename, "../templates/pages/import.html"),
	}...))

	expensesTempl = template.Must(template.Must(baseTempl.Clone()).ParseFiles([]string{
		filepath.Join(filename, "../templates/pages/expenses.html"),
	}...))

	categoriesTempl = template.Must(template.Must(baseTempl.Clone()).ParseFiles([]string{
		filepath.Join(filename, "../templates/pages/categories.html"),
	}...))

	newCategoriesTempl = template.Must(template.Must(baseTempl.Clone()).ParseFiles([]string{
		filepath.Join(filename, "../templates/pages/categories/new.html"),
	}...))

	newCategoryResult = template.Must(template.New("").Funcs(templateFuncs).ParseFiles([]string{
		filepath.Join(filename, "../templates/partials/categories/new_result.html"),
	}...))

	reportTempl = template.Must(template.New("").Funcs(templateFuncs).ParseFiles(
		filepath.Join(filename, "../templates/partials/reports/report.html")),
	)

	searchResultsTempl = template.Must(template.New("").Funcs(templateFuncs).ParseFiles(
		filepath.Join(filename, "../templates/partials/search/results.html")),
	)

	uncategoriesTempl = template.Must(template.New("").Funcs(templateFuncs).ParseFiles(
		filepath.Join(filename, "../templates/partials/categories/uncategorized.html")),
	)
}

func (router *router) parseTemplates() {
	if router.reload {
		parseLocalTemplates()
	} else {
		parseFSTemplates()
	}
}

func (router *router) liveReloadTemplatesMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if router.reload {
			router.parseTemplates()
		}
		next.ServeHTTP(w, r)
	})
}
