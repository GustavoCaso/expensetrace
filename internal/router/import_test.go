package router

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestImport(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	// Create test categories
	_, err := s.CreateCategory(context.Background(), user.ID(), "Food", "restaurant|food|grocery")
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	_, err = s.CreateCategory(context.Background(), user.ID(), "Transport", "uber|taxi|transit")
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	// Create test expenses
	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, storage.ChargeType, nil),
		storage.NewExpense(0, "Test Source", "Uber ride", "USD", -50000, now, storage.ChargeType, nil),
	}

	_, expenseError := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if expenseError != nil {
		t.Fatalf("Failed to insert test expenses: %v", expenseError)
	}

	// Create router
	handler, _ := New(s, logger)

	// Hit home to populate cache
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %v; got %v", http.StatusOK, resp.Status)
	}

	// Import new expenses
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// create a new form-data header name data and filename data.txt
	dataPart, err := writer.CreateFormFile("file", "expenses.json")
	if err != nil {
		t.Error(err.Error())
	}

	// copy file content into multipart section dataPart
	f, err := os.Open("test_data/import.json")
	if err != nil {
		t.Error(err.Error())
	}
	_, err = io.Copy(dataPart, f)
	if err != nil {
		t.Error(err.Error())
	}

	err = writer.Close()

	if err != nil {
		t.Error(err.Error())
	}

	req = httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %v; got %v", http.StatusOK, resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "import", resp.Body)
}

func ensureNoErrorInTemplateResponse(t *testing.T, test string, body io.ReadCloser) {
	t.Helper()

	byteResponse, err := io.ReadAll(body)

	if err != nil {
		t.Fatalf("error reading the response for '%s': %s", test, err.Error())
	}

	response := string(byteResponse)

	if strings.Contains(response, templateNotAvailableError) ||
		strings.Contains(response, templateRenderingError) {
		t.Fatalf("Error rendenring template for '%s' response: %s", test, response)
	}
}

// TestInteractiveImportPreview tests the preview step of interactive import.
func TestInteractiveImportPreview(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	// Create CSV data for upload
	csvData := `source,date,description,amount,currency
Bank A,01/01/2024,Coffee,-5.00,USD
Bank B,02/01/2024,Lunch,-12.00,USD`

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	dataPart, err := writer.CreateFormFile("file", "test.csv")
	if err != nil {
		t.Fatal(err)
	}

	_, err = dataPart.Write([]byte(csvData))
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %v; got %v", http.StatusOK, resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	responseBody := string(bodyBytes)

	// Check response contains expected elements
	if !strings.Contains(responseBody, "source") {
		t.Error("Response should contain 'source' header")
	}
	if !strings.Contains(responseBody, "Coffee") {
		t.Error("Response should contain 'Coffee' preview data")
	}
	if !strings.Contains(responseBody, "import_session_id") {
		t.Error("Response should contain import_session_id hidden field")
	}
}

// TestInteractiveImportInvalidFile tests error handling for invalid files.
func TestInteractiveImportInvalidFile(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	handler, _ := New(s, logger)

	// Upload file with unsupported format
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	dataPart, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatal(err)
	}

	_, err = dataPart.Write([]byte("some random text"))
	if err != nil {
		t.Fatal(err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %v; got %v", http.StatusOK, resp.Status)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	responseBody := string(bodyBytes)

	// Should contain error message
	if !strings.Contains(responseBody, "error") && !strings.Contains(responseBody, "Error") {
		t.Error("Response should contain error message for invalid file format")
	}
}
