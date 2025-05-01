package router

import (
	"bytes"
	"errors"
	"fmt"
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
		fmt.Fprint(w, "error r.ParseMultipartForm() ", err.Error())
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
		fmt.Fprint(w, errorMessage)
		return
	}
	defer file.Close()

	// Copy the file data to my buffer
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	log.Printf("Importing File name %s. Size %dKB\n", header.Filename, buf.Len())
	info := importUtil.Import(header.Filename, &buf, router.db, router.matcher)
	if info.Error != nil && info.TotalImports == 0 {
		fmt.Fprint(w, "Unable to import expenses due to error: ", info.Error)
		return
	}

	router.resetCache()

	fmt.Fprint(w, "Imported")
}
