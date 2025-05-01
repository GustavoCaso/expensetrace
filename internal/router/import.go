package router

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"

	importUtil "github.com/GustavoCaso/expensetrace/internal/import"
)

const (
	maxMemory = 32 << 20 // 32MB
)

func (router *router) importHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(maxMemory)

	if err != nil {
		data := struct {
			Error string
		}{
			Error: "Error parsing form: " + err.Error(),
		}

		router.templates.Render(w, "partials/import/result", data)
		return
	}

	file, header, err := r.FormFile("file")

	if err != nil {
		var errorMessage string
		if errors.Is(err, http.ErrMissingFile) {
			errorMessage = "No file submitted"
		} else {
			errorMessage = "Error retrieving the file"
		}
		data := struct {
			Error string
		}{
			Error: "Error parsing form: " + errorMessage,
		}

		router.templates.Render(w, "partials/import/result", data)
		return
	}
	defer file.Close()

	// Copy the file data to my buffer
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		data := struct {
			Error string
		}{
			Error: "Error parsing form: " + err.Error(),
		}

		router.templates.Render(w, "partials/import/result", data)
		return
	}
	log.Printf("Importing File name %s. Size %dKB\n", header.Filename, buf.Len())
	info := importUtil.Import(header.Filename, &buf, router.db, router.matcher)

	if info.Error != nil && info.TotalImports == 0 {
		data := struct {
			Error string
		}{
			Error: "Error importing expenses: " + info.Error.Error(),
		}

		router.templates.Render(w, "partials/import/result.html", data)
		return
	}

	router.templates.Render(w, "partials/import/result.html", info)

	// Reset cache to refresh data after import
	router.resetCache()
}
