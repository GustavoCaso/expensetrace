package web

import (
	"embed"
	"html/template"

	"github.com/GustavoCaso/expensetrace/internal/util"
)

// content holds our static content.
//
//go:embed templates
var templatesFS embed.FS

// Templates
var baseTempl *template.Template
var indexTempl *template.Template
var importTempl *template.Template
var searchResultsTempl *template.Template

var templateFuncs = template.FuncMap{
	"formatMoney": util.FormatMoney,
}

func parseTemplates() {
	baseTempl = template.Must(template.New("base").Funcs(templateFuncs).ParseFS(templatesFS, []string{
		"templates/home.html",
		"templates/partials/nav.html",
		"templates/partials/search.html",
	}...))

	indexTempl = template.Must(template.Must(baseTempl.Clone()).ParseFS(templatesFS, []string{
		"templates/pages/index.html",
	}...))

	importTempl = template.Must(template.Must(baseTempl.Clone()).ParseFS(templatesFS, []string{
		"templates/pages/import.html",
	}...))

	searchResultsTempl = template.Must(template.New("").Funcs(templateFuncs).ParseFS(templatesFS,
		"templates/partials/searchResults.html"),
	)
}
