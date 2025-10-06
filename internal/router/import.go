package router

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	importUtil "github.com/GustavoCaso/expensetrace/internal/import"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

const (
	maxMemory = 32 << 20 // 32MB
)

type importHandler struct {
	*router
	sessionStore *importUtil.SessionStore
}

func (i *importHandler) RegisterRoutes(mux *http.ServeMux) {
	// Initialize session store with 30 minute TTL
	const sessionTTL = 30 * time.Minute
	i.sessionStore = importUtil.NewSessionStore(sessionTTL)

	mux.HandleFunc("GET /import", func(w http.ResponseWriter, _ *http.Request) {
		data := viewBase{CurrentPage: pageImport}
		i.templates.Render(w, "pages/import/index.html", data)
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

const bytesPerKB = 1024

func (i *importHandler) importHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	data := viewBase{}
	data.CurrentPage = pageImport
	previewFlow := false

	defer func() {
		if !previewFlow {
			i.templates.Render(w, "partials/import/form.html", data)
		}
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

	// Copy the file data to my buffer
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		data.Error = fmt.Sprintf("Error copying bytes: %s", err.Error())
		return
	}

	sizeKB := fmt.Sprintf("%dKB", buf.Len()/bytesPerKB)
	i.logger.Info("File uploaded for import", "filename", header.Filename, "size", sizeKB)
	file.Close()

	fileFormat := path.Ext(header.Filename)
	if fileFormat == ".csv" {
		if !importUtil.SupportedProvider(header.Filename) {
			// Start prevideflow
			previewFlow = true
			i.previewHandler(ctx, &buf, header, w)
			return
		}
	}

	info := importUtil.Import(ctx, header.Filename, &buf, i.storage, i.matcher)

	if info.Error != nil && info.TotalImports == 0 {
		data.Error = fmt.Sprintf("Error importing expenses: %s", info.Error.Error())
		return
	}
	i.logger.Info("Imported succeeded 🎉", "total", info.TotalImports)

	var b strings.Builder
	fmt.Fprintf(&b, "%d expenses imported.", info.TotalImports)
	if info.TotalImports > 0 {
		fmt.Fprintf(&b, "%d expenses without category", len(info.ImportWithoutCategory))
	}

	banner := banner{
		Icon:    "✅",
		Message: b.String(),
	}
	data.Banner = banner

	// Reset cache to refresh data after import
	i.resetCache()
}

type previewData struct {
	viewBase
	ImportSessionID string
	Filename        string
	Headers         []string
	PreviewRows     [][]string
	TotalRows       int
}

const previewRowCount = 5

// previewHandler handles file upload and shows preview with column detection.
func (i *importHandler) previewHandler(
	_ context.Context,
	buf *bytes.Buffer,
	fileHeader *multipart.FileHeader,
	w http.ResponseWriter,
) {
	data := previewData{viewBase: viewBase{CurrentPage: pageImport}}

	defer func() {
		i.templates.Render(w, "partials/import/preview.html", data)
	}()

	// Parse the file
	parsedData, err := importUtil.ParseFile(fileHeader.Filename, buf)
	if err != nil {
		data.Error = fmt.Sprintf("Error parsing file: %s", err.Error())
		return
	}

	// Create session
	sessionID := i.sessionStore.Create(fileHeader.Filename, parsedData)

	data.ImportSessionID = sessionID
	data.Filename = fileHeader.Filename
	data.Headers = parsedData.Headers
	data.PreviewRows = parsedData.GetPreviewRows(previewRowCount)
	data.TotalRows = parsedData.GetTotalRows()
}

type mappingData struct {
	viewBase
	ImportSessionID string
	Headers         []string
	PreviewExpenses []storage.Expense
	TotalRows       int
	Errors          []string
}

// mappingHandler handles field mapping and shows confirmation preview.
func (i *importHandler) mappingHandler(_ context.Context, w http.ResponseWriter, r *http.Request) {
	data := mappingData{viewBase: viewBase{CurrentPage: pageImport}}

	defer func() {
		i.templates.Render(w, "partials/import/mapping-preview.html", data)
	}()

	// Get session ID from form
	sessionID := r.FormValue("import_session_id")
	if sessionID == "" {
		data.Error = "Session ID is required"
		return
	}

	// Retrieve session
	session, exists := i.sessionStore.Get(sessionID)
	if !exists {
		data.Error = "Session expired or not found. Please upload the file again."
		return
	}

	// Parse field mapping from form
	source := r.FormValue("source")
	if source == "" {
		data.Error = "Source is required"
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

	// Create mapping
	mapping := &importUtil.FieldMapping{
		Source:            source,
		DateColumn:        dateCol,
		DescriptionColumn: descCol,
		AmountColumn:      amountCol,
		CurrencyColumn:    currencyCol,
	}

	// Validate mapping
	if validationErr := mapping.Validate(len(session.Data.Headers)); validationErr != nil {
		data.Error = fmt.Sprintf("Invalid mapping: %s", validationErr.Error())
		return
	}

	// Apply mapping to first few rows for preview
	result, err := importUtil.ApplyMapping(session.Data, mapping, i.matcher)
	if err != nil {
		data.Error = fmt.Sprintf("Error applying mapping: %s", err.Error())
		return
	}

	// Store mapping in session
	i.sessionStore.Update(sessionID, mapping)

	// Prepare preview data (first 5 successfully mapped expenses)
	const maxPreviewExpenses = 5
	previewCount := min(maxPreviewExpenses, len(result.Expenses))

	previewExpenses := make([]storage.Expense, previewCount)
	for i := range previewCount {
		previewExpenses[i] = result.Expenses[i].Expense
	}

	// Collect error messages
	const headerRowOffset = 2
	errorMessages := make([]string, 0, len(result.Errors))
	for _, mappingErr := range result.Errors {
		rowNum := mappingErr.RowIndex + headerRowOffset
		errorMsg := fmt.Sprintf("Row %d: %s", rowNum, mappingErr.Error.Error())
		errorMessages = append(errorMessages, errorMsg)
	}

	data.ImportSessionID = sessionID
	data.Headers = session.Data.Headers
	data.PreviewExpenses = previewExpenses
	data.TotalRows = session.Data.GetTotalRows()
	data.Errors = errorMessages

	i.logger.Info(
		"Field mapping applied",
		"import_session_id", sessionID,
		"valid_rows", len(result.Expenses),
		"error_rows", len(result.Errors),
	)
}

// executeImportHandler executes the final import with stored mapping.
func (i *importHandler) executeImportHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	data := viewBase{}
	data.CurrentPage = pageImport

	defer func() {
		i.templates.Render(w, "partials/import/form.html", data)
	}()
	// Get session ID from form
	sessionID := r.FormValue("import_session_id")
	if sessionID == "" {
		data.Error = "Session ID is required"
		return
	}

	// Retrieve session
	session, exists := i.sessionStore.Get(sessionID)
	if !exists {
		data.Error = "Session expired or not found. Please upload the file again."
		return
	}

	// Check if mapping exists
	if session.Mapping == nil {
		data.Error = "No field mapping found. Please complete the mapping step first."
		return
	}

	i.logger.Info("Executing import", "import_session_id", sessionID, "filename", session.Filename)

	// Apply mapping to all rows
	result, err := importUtil.ApplyMapping(session.Data, session.Mapping, i.matcher)
	if err != nil {
		data.Error = fmt.Sprintf("Error applying mapping: %s", err.Error())
		return
	}

	// Convert mapped expenses to storage expenses
	expenses := make([]storage.Expense, len(result.Expenses))
	withoutCategory := make([]storage.Expense, 0)

	for i, mappedExp := range result.Expenses {
		expenses[i] = mappedExp.Expense
		if mappedExp.Category == "" {
			withoutCategory = append(withoutCategory, mappedExp.Expense)
		}
	}

	// Insert expenses
	inserted, err := i.storage.InsertExpenses(ctx, expenses)
	if err != nil {
		data.Error = fmt.Sprintf("Error inserting expenses: %s", err.Error())
		return
	}

	i.logger.Info(
		"Import completed successfully",
		"import_session_id", sessionID,
		"imported", inserted,
		"errors", len(result.Errors),
	)

	var b strings.Builder
	fmt.Fprintf(&b, "%d expenses imported.", int(inserted))
	if int(inserted) > 0 {
		fmt.Fprintf(&b, "%d expenses without category", len(withoutCategory))
	}

	banner := banner{
		Icon:    "✅",
		Message: b.String(),
	}
	data.Banner = banner

	// Clean up session
	i.sessionStore.Delete(sessionID)

	// Reset cache to refresh data
	i.resetCache()
}
