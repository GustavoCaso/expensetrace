package router

import (
	"embed"
	"encoding/json"
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
	"json": func(v interface{}) string {
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "[]"
		}
		return string(jsonBytes)
	},
}

type templates map[string]*template.Template

func (t templates) Render(w io.Writer, templateName string, data interface{}) {
	temp, ok := t[templateName]

	if !ok {
		_, _ = fmt.Fprintf(w, "template '%s' is not available", templateName)
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
		_, _ = fmt.Fprint(w, errorMessage)
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

func parseTemplates(fsDir fs.FS) (templates, error) {
	t := templates{}

	// First, collect all partials with proper naming
	partials := template.New("partials").Funcs(templateFuncs)
	err := fs.WalkDir(fsDir, "partials", func(filepath string, d fs.DirEntry, _ error) error {
		if !d.IsDir() {
			b, err := fs.ReadFile(fsDir, filepath)
			if err != nil {
				return err
			}

			// Use a template name that includes the path (excluding the "partials/" prefix)
			templateName := strings.TrimPrefix(filepath, "partials/")

			// Parse with the unique template name
			_, err = partials.New(templateName).Parse(string(b))
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return t, err
	}

	// Then create the base template with layout and add partials
	baseTempl := template.New("base").Funcs(templateFuncs)

	// Add all partials to the base template
	for _, template := range partials.Templates() {
		if template.Name() != "partials" { // Skip the root template
			_, err = baseTempl.AddParseTree(template.Name(), template.Tree)
			if err != nil {
				return t, err
			}
		}
	}

	// Parse layout files
	layoutBytes, err := fs.ReadFile(fsDir, "layout.html")
	if err != nil {
		return t, err
	}
	baseTempl, err = baseTempl.Parse(string(layoutBytes))
	if err != nil {
		return t, err
	}

	// Also parse nav partial which is needed by layout
	navBytes, err := fs.ReadFile(fsDir, "partials/nav.html")
	if err != nil {
		return t, err
	}
	baseTempl, err = baseTempl.Parse(string(navBytes))
	if err != nil {
		return t, err
	}

	// Parse pages with the enhanced base template
	err = fs.WalkDir(fsDir, "pages", func(path string, d fs.DirEntry, _ error) error {
		if !d.IsDir() {
			b, readErr := fs.ReadFile(fsDir, path)
			if readErr != nil {
				return readErr
			}

			pageTempl, cloneErr := baseTempl.Clone()
			if cloneErr != nil {
				return cloneErr
			}

			pageTempl, parseErr := pageTempl.Parse(string(b))
			if parseErr != nil {
				return parseErr
			}

			t[path] = pageTempl
		}
		return nil
	})

	if err != nil {
		return t, err
	}

	// Also add the partials as standalone templates (for direct rendering)
	_ = fs.WalkDir(fsDir, "partials", func(filepath string, d fs.DirEntry, _ error) error {
		if !d.IsDir() {
			templateName := strings.TrimPrefix(filepath, "partials/")
			partialTemplate := partials.Lookup(templateName)
			if partialTemplate != nil {
				t[filepath] = partialTemplate
			}
		}
		return nil
	})

	return t, nil
}

func (router *router) parseTemplates() error {
	var fs fs.FS

	if router.reload {
		fs = localFSDirectory()
	} else {
		fs = embeddedFS()
	}

	t, err := parseTemplates(fs)

	if err != nil {
		return err
	}

	router.templates = t
	return nil
}

type liveReloadTemplatesMiddleware struct {
	handler http.Handler
	router  *router
}

func (l *liveReloadTemplatesMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if l.router.reload {
		_ = l.router.parseTemplates()
	}
	l.handler.ServeHTTP(w, r)
}

func newLiveReloadMiddleware(router *router, handlder http.Handler) *liveReloadTemplatesMiddleware {
	return &liveReloadTemplatesMiddleware{
		router:  router,
		handler: handlder,
	}
}
