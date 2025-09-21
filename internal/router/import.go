package router

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	importUtil "github.com/GustavoCaso/expensetrace/internal/import"
)

const (
	maxMemory = 32 << 20 // 32MB
)

type importHandler struct {
	*router
}

func (i *importHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /import", func(w http.ResponseWriter, _ *http.Request) {
		data := viewBase{CurrentPage: pageImport}
		i.templates.Render(w, "pages/import/index.html", data)
	})

	mux.HandleFunc("POST /import", func(w http.ResponseWriter, r *http.Request) {
		i.importHandler(r.Context(), w, r)
	})
}

type importViewData struct {
	viewBase
	importUtil.ImportInfo
	Error string
}

func (i *importHandler) importHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	data := importViewData{}
	data.CurrentPage = pageImport

	defer func() {
		i.templates.Render(w, "partials/import/result.html", data)
	}()

	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		data.Error = fmt.Sprintf("Error parsing form: %s", err.Error())
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
		return
	}
	defer file.Close()

	// Copy the file data to my buffer
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		data.Error = fmt.Sprintf("Error copying bytes: %s", err.Error())
		return
	}
	i.logger.Info("Importing started", "file_name", header.Filename, "size", fmt.Sprintf("%dKB", buf.Len()))

	info := importUtil.Import(ctx, header.Filename, &buf, i.storage, i.matcher)

	if info.Error != nil && info.TotalImports == 0 {
		data.Error = fmt.Sprintf("Error importing expenses: %s", info.Error.Error())
		return
	}
	i.logger.Info("Imported succeeded 🎉", "total", info.TotalImports)

	data.ImportInfo = info

	// Reset cache to refresh data after import
	i.resetCache()
}
