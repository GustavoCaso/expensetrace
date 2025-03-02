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
	"path"
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

func parsePages(fsDir fs.FS, basetempl *template.Template, templates templates) {
	fs.WalkDir(fsDir, "pages", func(path string, d fs.DirEntry, err error) error {
		// If is not a dir, then we can assume that is the final html page template
		if !d.IsDir() {
			b, err := fs.ReadFile(fsDir, path)
			if err != nil {
				panic(err)
			}

			t := template.Must(template.Must(basetempl.Clone()).Parse(string(b)))
			// Store the new created template in the templates map
			templates[path] = t
		}
		return nil
	})
}

func parsePartials(fsDir fs.FS, templates templates) {
	fs.WalkDir(fsDir, "partials", func(filepath string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			name := path.Base(filepath)
			b, err := fs.ReadFile(fsDir, filepath)
			if err != nil {
				panic(err)
			}

			t := template.Must(template.New(name).Funcs(templateFuncs).Parse(string(b)))

			templates[filepath] = t
		}
		return nil
	})
}

func parseTemplates(fsDir fs.FS) templates {
	templates := templates{}

	baseTempl := template.Must(template.New("base").Funcs(templateFuncs).ParseFS(fsDir, []string{
		"layout.html",
		"partials/nav.html",
	}...))

	parsePages(fsDir, baseTempl, templates)

	parsePartials(fsDir, templates)

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
