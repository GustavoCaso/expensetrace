package importsvc

import (
	"context"
	"strings"
	"testing"
	"time"

	importUtil "github.com/GustavoCaso/expensetrace/import"
	"github.com/GustavoCaso/expensetrace/matcher"
	"github.com/GustavoCaso/expensetrace/testutil"
)

const testSessionTTL = 30 * time.Minute

const evoCSV = `Fecha de la operación,Fecha Valor,Concepto,Importe,Divisa,Tipo de movimiento,Saldo disponible
01/01/2024,,Restaurant bill,-1234.56,USD,,5000.00
02/01/2024,,Uber ride,-5000.00,USD,,0.00`

const genericCSV = `date,description,amount,currency
2024-01-01,Restaurant bill,-1234.56,USD
2024-01-02,Uber ride,-5000.00,USD`

// invalidSchemaJSON is syntactically valid JSON (an array of objects) but
// does not match importUtil.JSONExpense's required fields (missing
// "source", "currency", and "date"), so importUtil.SupportedJSONSchema
// rejects it and ImportFile must fall back to the interactive preview flow.
const invalidSchemaJSON = `[
  {"description": "Restaurant bill", "amount": -1234},
  {"description": "Uber ride", "amount": -5000}
]`

func TestImportFile_CSVKnownProvider(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger, testSessionTTL)
	m := matcher.New(nil)

	info, needsPreview, previewReader, err := svc.ImportFile(
		context.Background(),
		user.ID(),
		"evo_test.csv",
		strings.NewReader(evoCSV),
		m,
	)
	if err != nil {
		t.Fatalf("ImportFile returned error: %v", err)
	}

	if needsPreview {
		t.Fatal("Expected needsPreview=false for known provider")
	}

	if previewReader != nil {
		t.Fatal("Expected nil previewReader for known provider")
	}

	if info.TotalImports != 2 {
		t.Fatalf("Expected 2 imports, got %d", info.TotalImports)
	}

	allExpenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(allExpenses) != 2 {
		t.Fatalf("Expected 2 expenses in storage, got %d", len(allExpenses))
	}
}

func TestImportFile_UnsupportedCSVNeedsPreview(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger, testSessionTTL)
	m := matcher.New(nil)

	info, needsPreview, previewReader, err := svc.ImportFile(
		context.Background(),
		user.ID(),
		"unknown_format.csv",
		strings.NewReader(genericCSV),
		m,
	)
	if err != nil {
		t.Fatalf("ImportFile returned error: %v", err)
	}

	if !needsPreview {
		t.Fatal("Expected needsPreview=true for unsupported provider")
	}

	if previewReader == nil {
		t.Fatal("Expected non-nil previewReader for unsupported provider")
	}

	if (info != importUtil.ImportInfo{}) {
		t.Fatalf("Expected zero-value ImportInfo, got %+v", info)
	}

	allExpenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(allExpenses) != 0 {
		t.Fatalf("Expected 0 expenses in storage, got %d", len(allExpenses))
	}
}

func TestImportFile_InvalidJSONSchemaNeedsPreview(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger, testSessionTTL)
	m := matcher.New(nil)

	info, needsPreview, previewReader, err := svc.ImportFile(
		context.Background(),
		user.ID(),
		"invalid_schema.json",
		strings.NewReader(invalidSchemaJSON),
		m,
	)
	if err != nil {
		t.Fatalf("ImportFile returned error: %v", err)
	}

	if !needsPreview {
		t.Fatal("Expected needsPreview=true for invalid JSON schema")
	}

	if previewReader == nil {
		t.Fatal("Expected non-nil previewReader for invalid JSON schema")
	}

	if (info != importUtil.ImportInfo{}) {
		t.Fatalf("Expected zero-value ImportInfo, got %+v", info)
	}

	// The reader returned by ImportFile must be rewound to the start so
	// that Preview can read the full content again. If the Seek(0,
	// io.SeekStart) call were removed, this Preview call would read from
	// wherever SupportedJSONSchema's decoder left off (typically EOF),
	// and would either fail to parse or return no rows.
	headers, previewRows, totalRows, sessionID, err := svc.Preview("invalid_schema.json", previewReader)
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if sessionID == "" {
		t.Fatal("Expected non-empty sessionID from Preview")
	}

	if len(headers) == 0 {
		t.Fatal("Expected non-empty headers from Preview, indicating the reader was rewound correctly")
	}

	if totalRows != 2 {
		t.Fatalf("Expected 2 total rows, got %d", totalRows)
	}

	if len(previewRows) != 2 {
		t.Fatalf("Expected 2 preview rows, got %d", len(previewRows))
	}

	// Verify the actual content came through, not just row counts.
	found := false
	for _, row := range previewRows {
		for _, cell := range row {
			if cell == "Restaurant bill" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("Expected preview rows to contain 'Restaurant bill', got %+v", previewRows)
	}
}

func TestExecute_FailsWithoutMapping(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger, testSessionTTL)
	m := matcher.New(nil)

	_, _, _, sessionID, err := svc.Preview("unknown_format.csv", strings.NewReader(genericCSV))
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	_, _, _, err = svc.Execute(context.Background(), 0, sessionID, m)
	if err == nil {
		t.Fatal("Expected error when calling Execute without a mapping applied")
	}

	const expectedErrMsg = "No field mapping found. Please complete the mapping step first."
	if err.Error() != expectedErrMsg {
		t.Fatalf("Expected error message %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestApplyMapping_ReturnsPreviewRows(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, _ := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger, testSessionTTL)
	m := matcher.New(nil)

	_, _, _, sessionID, err := svc.Preview("unknown_format.csv", strings.NewReader(genericCSV))
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	mapping := &importUtil.FieldMapping{
		Source:            "MyBank",
		DateColumn:        0,
		DescriptionColumn: 1,
		AmountColumn:      2,
		CurrencyColumn:    3,
	}

	result, err := svc.ApplyMapping(sessionID, mapping, m)
	if err != nil {
		t.Fatalf("ApplyMapping returned error: %v", err)
	}

	if result.TotalRows != 2 {
		t.Fatalf("Expected TotalRows=2, got %d", result.TotalRows)
	}

	if len(result.PreviewExpenses) != 2 {
		t.Fatalf("Expected 2 preview expenses, got %d", len(result.PreviewExpenses))
	}

	if len(result.Errors) != 0 {
		t.Fatalf("Expected no mapping errors, got %v", result.Errors)
	}

	if result.PreviewExpenses[0].Description() != "restaurant bill" {
		t.Fatalf("Expected first preview expense description 'restaurant bill', got %s",
			result.PreviewExpenses[0].Description())
	}
}

func TestExecute_InsertsMappedExpenses(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger, testSessionTTL)
	m := matcher.New(nil)

	_, _, _, sessionID, err := svc.Preview("unknown_format.csv", strings.NewReader(genericCSV))
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	mapping := &importUtil.FieldMapping{
		Source:            "MyBank",
		DateColumn:        0,
		DescriptionColumn: 1,
		AmountColumn:      2,
		CurrencyColumn:    3,
	}

	_, err = svc.ApplyMapping(sessionID, mapping, m)
	if err != nil {
		t.Fatalf("ApplyMapping returned error: %v", err)
	}

	inserted, withoutCategory, resultErrorsCount, err := svc.Execute(context.Background(), user.ID(), sessionID, m)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if inserted != 2 {
		t.Fatalf("Expected 2 inserted, got %d", inserted)
	}

	if withoutCategory != 2 {
		t.Fatalf("Expected 2 without category, got %d", withoutCategory)
	}

	if resultErrorsCount != 0 {
		t.Fatalf("Expected 0 result errors, got %d", resultErrorsCount)
	}

	allExpenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(allExpenses) != 2 {
		t.Fatalf("Expected 2 expenses in storage, got %d", len(allExpenses))
	}

	// A second Execute should fail since the session was deleted.
	_, _, _, err = svc.Execute(context.Background(), user.ID(), sessionID, m)
	if err == nil {
		t.Fatal("Expected error on second Execute call after session deletion")
	}
}
