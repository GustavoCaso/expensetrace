package router

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/GustavoCaso/expensetrace/internal/util"
)

// content holds our static content.
//
//go:embed templates
var templatesFS embed.FS

var templateFuncs = template.FuncMap{
	"formatMoney": util.FormatMoney,
}

type templates struct {
	indexTempl         *template.Template
	reportTempl        *template.Template
	importTempl        *template.Template
	searchResultsTempl *template.Template
	expensesTempl      *template.Template

	categoriesTempl    *template.Template
	uncategoriesTempl  *template.Template
	newCategoriesTempl *template.Template
	newCategoryResult  *template.Template
}

func localFSDirectory() fs.FS {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Printf("enable to get current directory %s. defaulting to embedded templates\n", filename)
		return embeddedFS()
	}

	return os.DirFS(filepath.Join(filename, "../templates"))
}

func embeddedFS() fs.FS {
	subTemplateFS, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		panic(err)
	}

	return subTemplateFS
}

func parseTemplates(fsDir fs.FS) *templates {
	baseTempl := template.Must(template.New("base").Funcs(templateFuncs).ParseFS(fsDir, []string{
		"home.html",
		"partials/nav.html",
		"partials/search/form.html",
	}...))

	indexTempl := template.Must(template.Must(baseTempl.Clone()).ParseFS(fsDir, []string{
		"pages/index.html",
	}...))

	importTempl := template.Must(template.Must(baseTempl.Clone()).ParseFS(fsDir, []string{
		"pages/import.html",
	}...))

	expensesTempl := template.Must(template.Must(baseTempl.Clone()).ParseFS(fsDir, []string{
		"pages/expenses.html",
	}...))

	categoriesTempl := template.Must(template.Must(baseTempl.Clone()).ParseFS(fsDir, []string{
		"pages/categories.html",
	}...))

	newCategoriesTempl := template.Must(template.Must(baseTempl.Clone()).ParseFS(fsDir, []string{
		"pages/categories/new.html",
	}...))

	newCategoryResult := template.Must(template.New("").Funcs(templateFuncs).ParseFS(fsDir, []string{
		"partials/categories/new_result.html",
	}...))

	reportTempl := template.Must(template.New("").Funcs(templateFuncs).ParseFS(fsDir,
		"partials/reports/report.html",
	))

	searchResultsTempl := template.Must(template.New("").Funcs(templateFuncs).ParseFS(fsDir,
		"partials/search/results.html"),
	)

	uncategoriesTempl := template.Must(template.New("").Funcs(templateFuncs).ParseFS(fsDir,
		"partials/categories/uncategorized.html"),
	)

	return &templates{
		indexTempl:         indexTempl,
		reportTempl:        reportTempl,
		importTempl:        importTempl,
		searchResultsTempl: searchResultsTempl,
		expensesTempl:      expensesTempl,
		categoriesTempl:    categoriesTempl,
		uncategoriesTempl:  uncategoriesTempl,
		newCategoriesTempl: newCategoriesTempl,
		newCategoryResult:  newCategoryResult,
	}
}

func (router *router) parseTemplates() {
	var fs fs.FS

	if router.reload {
		fs = localFSDirectory()
	} else {
		fs = embeddedFS()
	}

	router.templates = parseTemplates(fs)
}

type liveReloadTemplatesMiddleware struct {
	handler http.Handler
	router  *router
}

func (l *liveReloadTemplatesMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if l.router.reload {
		l.router.parseTemplates()
	}
	l.handler.ServeHTTP(w, r)
}

func newLiveReloadMiddleware(router *router, handlder http.Handler) *liveReloadTemplatesMiddleware {
	return &liveReloadTemplatesMiddleware{
		router:  router,
		handler: handlder,
	}
}
