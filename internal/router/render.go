package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/util"
)

type htmlRenderer struct {
	templateFS      fs.FS
	sharedTemplates *template.Template
}

// The newHTMLRenderer function creates a new htmlRenderer containing a shared
// set of parsed templates with support for any custom template functions.
func newHTMLRenderer(templateFS fs.FS, sharedTemplateFiles ...string) (*htmlRenderer, error) {
	funcs := template.FuncMap{
		"formatMoney": util.FormatMoney,
		"colorOutput": util.ColorOutput,
		"sub": func(a, b int) int {
			return a - b
		},
		"divideFloat": func(a int64, b int64) float64 {
			return float64(a) / float64(b)
		},
		"json": func(v any) string {
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return "[]"
			}
			return string(jsonBytes)
		},
		"divCents": func(cents int64) string {
			if cents == 0 {
				return ""
			}
			euros := float64(cents) / 100 //nolint:mnd // the value is obvious
			return fmt.Sprintf("%.2f", euros)
		},
		"amountToDollars": func(cents *int64) string {
			if cents == nil {
				return ""
			}
			dollars := float64(*cents) / 100.0 //nolint:mnd // the value is obvious
			return fmt.Sprintf("%.2f", dollars)
		},
		"formatDate": func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.Format("2006-01-02")
		},
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
	}

	sharedTemplates, err := template.New("").Funcs(funcs).ParseFS(templateFS, sharedTemplateFiles...)
	if err != nil {
		return nil, err
	}

	r := &htmlRenderer{
		templateFS:      templateFS,
		sharedTemplates: sharedTemplates,
	}

	return r, nil
}

// The render method clones the shared template set, optionally parses additional
// templates, executes the named template with the supplied data, and writes the
// response.
func (h *htmlRenderer) render(
	w http.ResponseWriter,
	status int,
	data any,
	templateName string,
	additionalTemplateFiles ...string,
) error {
	ts, err := h.sharedTemplates.Clone()
	if err != nil {
		return err
	}

	if len(additionalTemplateFiles) > 0 {
		ts, err = ts.ParseFS(h.templateFS, additionalTemplateFiles...)
		if err != nil {
			return err
		}
	}

	buf := new(bytes.Buffer)

	err = ts.ExecuteTemplate(buf, templateName, data)
	if err != nil {
		return err
	}

	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)

	return nil
}
