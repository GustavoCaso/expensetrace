// Package importsvc contains the business logic for the multi-step
// (upload -> preview -> map -> execute) expense import flow.
package importsvc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/domain"
	importUtil "github.com/GustavoCaso/expensetrace/internal/import"
	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

// previewExpenseCount is the number of rows shown when previewing a file
// upload or a field mapping.
const previewExpenseCount = 5

// headerRowOffset accounts for the header row when reporting row numbers in
// mapping error messages.
const headerRowOffset = 1

const bytesPerKB = 1024

type Service struct {
	storage      storage.Storage
	logger       *logger.Logger
	sessionStore *importUtil.SessionStore
}

func New(storage storage.Storage, logger *logger.Logger, sessionTTL time.Duration) *Service {
	return &Service{
		storage:      storage,
		logger:       logger,
		sessionStore: importUtil.NewSessionStore(sessionTTL),
	}
}

// ImportFile dispatches an uploaded file for import based on its extension.
//
// For CSV files from a recognized provider, and for JSON files matching the
// supported schema, the import is performed immediately and info is
// returned with needsPreview=false.
//
// Otherwise, needsPreview is true and previewReader holds the (possibly
// rewound) file contents ready to be passed to Preview.
func (s *Service) ImportFile(
	ctx context.Context,
	userID int64,
	filename string,
	r io.Reader,
	m *matcher.Matcher,
) (importUtil.ImportInfo, bool, io.Reader, error) {
	fileExtension := path.Ext(filename)

	switch fileExtension {
	case ".csv":
	case ".json":
	default:
		//nolint:staticcheck // preserves original user-facing message text
		return importUtil.ImportInfo{}, false, nil, fmt.Errorf("Error: unsupported file extesion: %s", fileExtension)
	}

	// Copy the file data to my buffer
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		//nolint:staticcheck // preserves original user-facing message text
		return importUtil.ImportInfo{}, false, nil, fmt.Errorf("Error copying bytes: %w", err)
	}

	sizeKB := fmt.Sprintf("%dKB", buf.Len()/bytesPerKB)
	s.logger.Info("File uploaded for import", "filename", filename, "size", sizeKB)

	if fileExtension == ".csv" {
		if !importUtil.SupportedProvider(filename) {
			// Start interactive flow
			return importUtil.ImportInfo{}, true, &buf, nil
		}

		info := importUtil.ImportCSV(ctx, userID, filename, &buf, s.storage, m)
		return info, false, nil, nil
	}

	// .json
	reader := bytes.NewReader(buf.Bytes())
	valid, jsonExpenses := importUtil.SupportedJSONSchema(reader)

	if !valid {
		// Rewind reader
		_, seekErr := reader.Seek(0, io.SeekStart)
		if seekErr != nil {
			//nolint:staticcheck // preserves original user-facing message text
			return importUtil.ImportInfo{}, false, nil, fmt.Errorf("Error occurred when reading the file: %w", seekErr)
		}
		// Start interactive flow
		return importUtil.ImportInfo{}, true, reader, nil
	}

	info := importUtil.ImportJSON(ctx, userID, jsonExpenses, s.storage, m)
	return info, false, nil, nil
}

// Preview parses a file and creates an import session, returning enough
// information to render a preview of its contents.
func (s *Service) Preview(filename string, r io.Reader) (
	[]string,
	[][]string,
	int,
	string,
	error,
) {
	parsedData, err := importUtil.ParseFile(filename, r)
	if err != nil {
		return nil, nil, 0, "", err
	}

	sessionID := s.sessionStore.Create(filename, parsedData)

	headers := parsedData.Headers
	previewRows := parsedData.GetPreviewRows(previewExpenseCount)
	totalRows := parsedData.GetTotalRows()

	return headers, previewRows, totalRows, sessionID, nil
}

// MappingApplication contains everything a caller needs to render the
// mapping-preview step after applying a field mapping to an import session.
type MappingApplication struct {
	Headers         []string
	PreviewExpenses []domain.Expense
	TotalRows       int
	Errors          []string
}

// ApplyMapping validates that the given import session exists, applies the
// field mapping to its parsed data, stores the mapping on the session, and
// returns the data needed to render a mapping preview.
func (s *Service) ApplyMapping(
	sessionID string,
	mapping *importUtil.FieldMapping,
	m *matcher.Matcher,
) (MappingApplication, error) {
	session, exists := s.sessionStore.Get(sessionID)
	if !exists {
		return MappingApplication{}, errSessionNotFound
	}

	result, err := importUtil.ApplyMapping(session.Data, mapping, m)
	if err != nil {
		//nolint:staticcheck // preserves original user-facing message text
		return MappingApplication{}, fmt.Errorf("Error applying mapping: %w", err)
	}

	s.sessionStore.Update(sessionID, mapping)

	previewCount := min(previewExpenseCount, len(result.Expenses))

	previewExpenses := make([]domain.Expense, previewCount)
	for i := range previewCount {
		previewExpenses[i] = result.Expenses[i]
	}

	errorMessages := make([]string, 0, len(result.Errors))
	for _, mappingErr := range result.Errors {
		rowNum := mappingErr.RowIndex + headerRowOffset
		errorMsg := fmt.Sprintf("Row %d: %s", rowNum, mappingErr.Error.Error())
		errorMessages = append(errorMessages, errorMsg)
	}

	s.logger.Info(
		"Field mapping applied",
		"import_session_id", sessionID,
		"valid_rows", len(result.Expenses),
		"error_rows", len(result.Errors),
	)

	return MappingApplication{
		Headers:         session.Data.Headers,
		PreviewExpenses: previewExpenses,
		TotalRows:       session.Data.GetTotalRows(),
		Errors:          errorMessages,
	}, nil
}

// errSessionNotFound is returned when an import session is missing or has
// expired.
//
//nolint:staticcheck,revive // preserves original user-facing message text
var errSessionNotFound = errors.New("Session expired or not found. Please upload the file again.")

// errNoMapping is returned when Execute is called on a session that has not
// yet had a field mapping applied.
//
//nolint:staticcheck,revive // preserves original user-facing message text
var errNoMapping = errors.New("No field mapping found. Please complete the mapping step first.")

// Execute applies the stored field mapping for the given import session and
// inserts the resulting expenses, then deletes the session.
func (s *Service) Execute(
	ctx context.Context,
	userID int64,
	sessionID string,
	m *matcher.Matcher,
) (int64, int, int, error) {
	session, exists := s.sessionStore.Get(sessionID)
	if !exists {
		return 0, 0, 0, errSessionNotFound
	}

	if session.Mapping == nil {
		return 0, 0, 0, errNoMapping
	}

	s.logger.Info("Executing import", "import_session_id", sessionID, "filename", session.Filename)

	result, err := importUtil.ApplyMapping(session.Data, session.Mapping, m)
	if err != nil {
		//nolint:staticcheck // preserves original user-facing message text
		return 0, 0, 0, fmt.Errorf("Error applying mapping: %w", err)
	}

	withoutCategory := 0
	for _, mappedExp := range result.Expenses {
		if mappedExp.CategoryID() == nil {
			withoutCategory++
		}
	}

	inserted, err := s.storage.InsertExpenses(ctx, userID, result.Expenses)
	if err != nil {
		//nolint:staticcheck // preserves original user-facing message text
		return 0, 0, 0, fmt.Errorf("Error inserting expenses: %w", err)
	}

	s.logger.Info(
		"Import completed successfully",
		"import_session_id", sessionID,
		"imported", inserted,
		"errors", len(result.Errors),
	)

	s.sessionStore.Delete(sessionID)

	return inserted, withoutCategory, len(result.Errors), nil
}
