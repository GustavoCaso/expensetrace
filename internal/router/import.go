package router

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/GustavoCaso/expensetrace/internal/domain"
	importUtil "github.com/GustavoCaso/expensetrace/internal/import"
)

const (
	maxMemory = 5 << 20 // 5MB
)

type importHandler struct {
	*router
}

func (i *importHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /import", func(w http.ResponseWriter, r *http.Request) {
		base := viewBaseFromContext(r.Context())
		i.renderHTML(w, http.StatusOK, base, "base", "pages/import/index.html")
	})

	mux.HandleFunc("POST /import", func(w http.ResponseWriter, r *http.Request) {
		i.importHandler(r.Context(), w, r)
	})

	mux.HandleFunc("POST /import/map", func(w http.ResponseWriter, r *http.Request) {
		i.mappingHandler(r.Context(), w, r)
	})

	mux.HandleFunc("POST /import/execute", func(w http.ResponseWriter, r *http.Request) {
		i.executeImportHandler(r.Context(), w, r)
	})
}

func (i *importHandler) importHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	data := viewBaseFromContext(ctx)
	previewFlow := false

	defer func() {
		if !previewFlow {
			i.renderHTML(w, http.StatusOK, data, "import/form")
		}
	}()

	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)
	err := r.ParseMultipartForm(maxMemory) //nolint:gosec // MaxBytesReader applied above
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

	categoryMatcher, err := i.categoryMatcher(ctx, userID)
	if err != nil {
		data.Error = err.Error()
		return
	}

	info, needsPreview, previewReader, err := i.importService.ImportFile(
		ctx,
		userID,
		header.Filename,
		file,
		categoryMatcher,
	)
	if err != nil {
		data.Error = err.Error()
		return
	}

	if needsPreview {
		previewFlow = true
		i.previewHandler(header.Filename, previewReader, w)
		return
	}

	if info.Error != nil && info.TotalImports == 0 {
		data.Error = fmt.Sprintf("Error importing expenses: %s", info.Error.Error())
		return
	}

	i.logger.Info("Imported succeeded 🎉", "total", info.TotalImports)

	var b strings.Builder
	fmt.Fprintf(&b, "%d expenses imported.", info.TotalImports)
	if info.TotalImports > 0 {
		fmt.Fprintf(&b, "%d expenses without category", info.ImportWithoutCategory)
	}

	banner := domain.Banner{
		Icon:    "✅",
		Message: b.String(),
	}
	data.Banner = banner
}

// previewHandler handles file upload and shows preview with column detection.
func (i *importHandler) previewHandler(
	filename string,
	reader io.Reader,
	w http.ResponseWriter,
) {
	data := domain.PreviewData{ViewBase: domain.ViewBase{CurrentPage: pageImport, LoggedIn: true}}

	defer func() {
		i.renderHTML(w, http.StatusOK, data, "import/preview")
	}()

	headers, previewRows, totalRows, sessionID, err := i.importService.Preview(filename, reader)
	if err != nil {
		data.Error = fmt.Sprintf("Error parsing file: %s", err.Error())
		return
	}

	data.ImportSessionID = sessionID
	data.Filename = filename
	data.Headers = headers
	data.PreviewRows = previewRows
	data.TotalRows = totalRows
}

// mappingHandler handles field mapping and shows confirmation preview.
func (i *importHandler) mappingHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	data := domain.MappingData{ViewBase: domain.ViewBase{CurrentPage: pageImport}}

	defer func() {
		i.renderHTML(w, http.StatusOK, data, "import/mapping-preview")
	}()

	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)
	if err := r.ParseForm(); err != nil {
		data.Error = fmt.Sprintf("Error parsing form: %s", err.Error())
		return
	}

	sessionID := r.FormValue("import_session_id")
	if sessionID == "" {
		data.Error = "Session ID is required"
		return
	}

	source := r.FormValue("source")
	if source == "" {
		data.Error = sourceIsRequired
		return
	}

	dateCol, err := strconv.Atoi(r.FormValue("date_column"))
	if err != nil {
		data.Error = "Invalid date column"
		return
	}

	descCol, err := strconv.Atoi(r.FormValue("description_column"))
	if err != nil {
		data.Error = "Invalid description column"
		return
	}

	amountCol, err := strconv.Atoi(r.FormValue("amount_column"))
	if err != nil {
		data.Error = "Invalid amount column"
		return
	}

	currencyCol, err := strconv.Atoi(r.FormValue("currency_column"))
	if err != nil {
		data.Error = "Invalid currency column"
		return
	}

	mapping := &importUtil.FieldMapping{
		Source:            source,
		DateColumn:        dateCol,
		DescriptionColumn: descCol,
		AmountColumn:      amountCol,
		CurrencyColumn:    currencyCol,
	}

	categoryMatcher, err := i.categoryMatcher(ctx, userIDFromContext(ctx))
	if err != nil {
		data.Error = err.Error()
		return
	}

	result, err := i.importService.ApplyMapping(sessionID, mapping, categoryMatcher)
	if err != nil {
		data.Error = err.Error()
		return
	}

	data.ImportSessionID = sessionID
	data.Headers = result.Headers
	data.PreviewExpenses = result.PreviewExpenses
	data.TotalRows = result.TotalRows
	data.Errors = result.Errors
}

// executeImportHandler executes the final import with stored mapping.
func (i *importHandler) executeImportHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	data := domain.ViewBase{}
	data.CurrentPage = pageImport
	data.LoggedIn = true

	defer func() {
		i.renderHTML(w, http.StatusOK, data, "import/form")
	}()

	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)
	if err := r.ParseForm(); err != nil {
		data.Error = fmt.Sprintf("Error parsing form: %s", err.Error())
		return
	}

	sessionID := r.FormValue("import_session_id")
	if sessionID == "" {
		data.Error = "Session ID is required"
		return
	}

	categoryMatcher, err := i.categoryMatcher(ctx, userID)
	if err != nil {
		data.Error = err.Error()
		return
	}

	inserted, withoutCategory, _, err := i.importService.Execute(ctx, userID, sessionID, categoryMatcher)
	if err != nil {
		data.Error = err.Error()
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d expenses imported.", int(inserted))
	if int(inserted) > 0 {
		fmt.Fprintf(&b, "%d expenses without category", withoutCategory)
	}

	banner := domain.Banner{
		Icon:    "✅",
		Message: b.String(),
	}
	data.Banner = banner
}
