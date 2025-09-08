package router

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

const templateNotAvailableError = "template is not available"
const templateRenderingError = "error rendering template"

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
	"divideFloat": func(a int64, b int64) float64 {
		return float64(a) / float64(b)
	},
	"json": func(v interface{}) string {
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "[]"
		}
		return string(jsonBytes)
	},
}

type templates struct {
	t      map[string]*template.Template
	logger *logger.Logger
}

// Render function writes the content of the template and data into `w`
// even if there is an error we write the error message into `w`
// If the writer is an http.ResponseWritter even when there is an error,
// the response woud be 200 as writing to the response writer set the status code to 200.
func (t *templates) Render(w io.Writer, templateName string, data interface{}) {
	temp, ok := t.t[templateName]

	if !ok {
		_, _ = fmt.Fprintf(w, "%s '%s'", templateNotAvailableError, templateName)
		return
	}
	prettyData, _ := json.MarshalIndent(data, "", " ")
	t.logger.Debug("Rendering template", "name", templateName, "data", prettyData)
	var err error

	if strings.Contains(templateName, "partials") {
		tName := strings.TrimSuffix(temp.Name(), ".html")
		err = temp.ExecuteTemplate(w, tName, data)
	} else {
		err = temp.Execute(w, data)
	}

	if err != nil {
		t.logger.Error("Template execution failed", "error", err)
		errorMessage := fmt.Sprintf("%s '%s': %v", templateRenderingError, templateName, err.Error())
		_, _ = fmt.Fprint(w, errorMessage)
	}
}

func localFSDirectory(logger *logger.Logger) (fs.FS, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		logger.Warn("Unable to get current directory, defaulting to embedded templates", "filename", filename)
		return embeddedFS()
	}

	return os.DirFS(filepath.Join(filename, "../templates")), nil
}

func embeddedFS() (fs.FS, error) {
	subTemplateFS, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, err
	}

	return subTemplateFS, nil
}

func (t *templates) parseTemplates(fsDir fs.FS) error {
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
		return err
	}

	// Then create the base template with layout and add partials
	baseTempl := template.New("base").Funcs(templateFuncs)

	// Add all partials to the base template
	for _, template := range partials.Templates() {
		if template.Name() != "partials" { // Skip the root template
			_, err = baseTempl.AddParseTree(template.Name(), template.Tree)
			if err != nil {
				return err
			}
		}
	}

	// Parse layout files
	layoutBytes, err := fs.ReadFile(fsDir, "layout.html")
	if err != nil {
		return err
	}
	baseTempl, err = baseTempl.Parse(string(layoutBytes))
	if err != nil {
		return err
	}

	// Also parse nav partial which is needed by layout
	navBytes, err := fs.ReadFile(fsDir, "partials/nav.html")
	if err != nil {
		return err
	}
	baseTempl, err = baseTempl.Parse(string(navBytes))
	if err != nil {
		return err
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

			t.t[path] = pageTempl
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Also add the partials as standalone templates (for direct rendering)
	_ = fs.WalkDir(fsDir, "partials", func(filepath string, d fs.DirEntry, _ error) error {
		if !d.IsDir() {
			templateName := strings.TrimPrefix(filepath, "partials/")
			partialTemplate := partials.Lookup(templateName)
			if partialTemplate != nil {
				t.t[filepath] = partialTemplate
			}
		}
		return nil
	})

	return nil
}
