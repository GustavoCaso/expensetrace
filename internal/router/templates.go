package router

import (
	"embed"
	"fmt"
	"html/template"
	"io"
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

type templates map[string]*template.Template

func (t templates) Render(w io.Writer, templateName string, data interface{}) error {
	temp, ok := t[templateName]

	if !ok {
		return fmt.Errorf("template '%s' is not available", templateName)
	}
	log.Printf("rendering template `%s`\n", templateName)
	return temp.Execute(w, data)
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

func parseTemplates(fsDir fs.FS) templates {
	templates := templates{}

	baseTempl := template.Must(template.New("base").Funcs(templateFuncs).ParseFS(fsDir, []string{
		"home.html",
		"partials/nav.html",
		"partials/search/form.html",
	}...))

	indexTempl := template.Must(template.Must(baseTempl.Clone()).ParseFS(fsDir, []string{
		"pages/index.html",
	}...))

	templates["pages/index.html"] = indexTempl

	importTempl := template.Must(template.Must(baseTempl.Clone()).ParseFS(fsDir, []string{
		"pages/import.html",
	}...))

	templates["pages/import.html"] = importTempl

	expensesTempl := template.Must(template.Must(baseTempl.Clone()).ParseFS(fsDir, []string{
		"pages/expenses.html",
	}...))

	templates["pages/expenses.html"] = expensesTempl

	categoriesTempl := template.Must(template.Must(baseTempl.Clone()).ParseFS(fsDir, []string{
		"pages/categories.html",
	}...))

	templates["pages/categories.html"] = categoriesTempl

	newCategoriesTempl := template.Must(template.Must(baseTempl.Clone()).ParseFS(fsDir, []string{
		"pages/categories/new.html",
	}...))

	templates["pages/categories/new.html"] = newCategoriesTempl

	newCategoryResult := template.Must(template.New("new_result.html").Funcs(templateFuncs).ParseFS(fsDir, []string{
		"partials/categories/new_result.html",
	}...))

	templates["partials/categories/new_result.html"] = newCategoryResult

	reportTempl := template.Must(template.New("report.html").Funcs(templateFuncs).ParseFS(fsDir,
		"partials/reports/report.html",
	))

	templates["partials/reports/report.html"] = reportTempl

	searchResultsTempl := template.Must(template.New("results.html").Funcs(templateFuncs).ParseFS(fsDir,
		"partials/search/results.html"),
	)

	templates["partials/search/results.html"] = searchResultsTempl

	uncategoriesTempl := template.Must(template.New("uncategorized.html").Funcs(templateFuncs).ParseFS(fsDir,
		"partials/categories/uncategorized.html"),
	)

	templates["partials/categories/uncategorized.html"] = uncategoriesTempl

	return templates
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
