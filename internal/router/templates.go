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
	"strings"

	"github.com/GustavoCaso/expensetrace/internal/util"
)

// content holds our static content.
//
//go:embed templates
var templatesFS embed.FS

var templateFuncs = template.FuncMap{
	"formatMoney": util.FormatMoney,
	"colorOutput": util.ColorOutput,
	"sub": func(a, b int) int {
		return a - b
	},
}

type templates map[string]*template.Template

func (t templates) Render(w io.Writer, templateName string, data interface{}) {
	temp, ok := t[templateName]

	if !ok {
		w.Write([]byte(fmt.Sprintf("template '%s' is not available", templateName)))
		return
	}
	log.Printf("rendering template `%s`\n", templateName)
	var err error
	if strings.Contains(templateName, "partials") {
		tName := strings.TrimSuffix(temp.Name(), ".html")
		err = temp.ExecuteTemplate(w, tName, data)
	} else {
		err = temp.Execute(w, data)
	}
	if err != nil {
		log.Print(err.Error())
		errorMessage := fmt.Sprintf("Error rendering template '%s': %v", templateName, err.Error())
		w.Write([]byte(errorMessage))
	}
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

	// First, collect all partials with proper naming
	partials := template.New("partials").Funcs(templateFuncs)
	fs.WalkDir(fsDir, "partials", func(filepath string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			b, err := fs.ReadFile(fsDir, filepath)
			if err != nil {
				panic(err)
			}

			// Use a template name that includes the path (excluding the "partials/" prefix)
			templateName := strings.TrimPrefix(filepath, "partials/")

			// Parse with the unique template name
			_, err = partials.New(templateName).Parse(string(b))
			if err != nil {
				panic(err)
			}
		}
		return nil
	})

	// Then create the base template with layout and add partials
	baseTempl := template.New("base").Funcs(templateFuncs)

	// Add all partials to the base template
	for _, t := range partials.Templates() {
		if t.Name() != "partials" { // Skip the root template
			_, err := baseTempl.AddParseTree(t.Name(), t.Tree)
			if err != nil {
				panic(err)
			}
		}
	}

	// Parse layout files
	layoutBytes, err := fs.ReadFile(fsDir, "layout.html")
	if err != nil {
		panic(err)
	}
	baseTempl, err = baseTempl.Parse(string(layoutBytes))
	if err != nil {
		panic(err)
	}

	// Also parse nav partial which is needed by layout
	navBytes, err := fs.ReadFile(fsDir, "partials/nav.html")
	if err != nil {
		panic(err)
	}
	baseTempl, err = baseTempl.Parse(string(navBytes))
	if err != nil {
		panic(err)
	}

	// Parse pages with the enhanced base template
	fs.WalkDir(fsDir, "pages", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			b, err := fs.ReadFile(fsDir, path)
			if err != nil {
				panic(err)
			}

			pageTempl, err := baseTempl.Clone()
			if err != nil {
				panic(err)
			}

			pageTempl, err = pageTempl.Parse(string(b))
			if err != nil {
				panic(err)
			}

			templates[path] = pageTempl
		}
		return nil
	})

	// Also add the partials as standalone templates (for direct rendering)
	fs.WalkDir(fsDir, "partials", func(filepath string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			templateName := strings.TrimPrefix(filepath, "partials/")
			partialTemplate := partials.Lookup(templateName)
			if partialTemplate != nil {
				templates[filepath] = partialTemplate
			}
		}
		return nil
	})

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
