package router

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

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
	errors := importUtil.Import(header.Filename, &buf, router.db, router.matcher)
	if len(errors) > 0 {
		errorStrings := make([]string, len(errors))
		for i, err := range errors {
			errorStrings[i] = err.Error()
		}
		errorMessage := strings.Join(errorStrings, "\n")
		log.Printf("Errors importing file: %s. %s", header.Filename, errorMessage)
		fmt.Fprint(w, errorMessage)
		return
	}

	router.resetCache()

	fmt.Fprint(w, "Imported")
}
