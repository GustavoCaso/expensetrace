package router

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	importUtil "github.com/GustavoCaso/expensetrace/internal/import"
)

const (
	maxMemory = 32 << 20 // 32MB
)

type importViewData struct {
	viewBase
	importUtil.ImportInfo
	Error string
}

func (router *router) importHandler(w http.ResponseWriter, r *http.Request) {
	data := importViewData{}
	err := r.ParseMultipartForm(maxMemory)

	if err != nil {
		data.Error = fmt.Sprintf("Error parsing form: %s", err.Error())

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
		data.Error = fmt.Sprintf("Error parsing form: %s", errorMessage)

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
	router.logger.Info(fmt.Sprintf("Importing File name %s. Size %dKB\n", header.Filename, buf.Len()))
	info := importUtil.Import(header.Filename, &buf, router.storage, router.matcher)

	if info.Error != nil && info.TotalImports == 0 {
		data.Error = fmt.Sprintf("Error importing expenses: %s", info.Error.Error())

		router.templates.Render(w, "partials/import/result.html", data)
		return
	}

	data.ImportInfo = info

	router.templates.Render(w, "partials/import/result.html", info)

	// Reset cache to refresh data after import
	router.resetCache()
}
