package router

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestImport(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	// Create test categories
	_, err := s.CreateCategory("Food", "restaurant|food|grocery")
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	_, err = s.CreateCategory("Transport", "uber|taxi|transit")
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	categories, err := s.GetCategories()
	if err != nil {
		t.Fatalf("Failed to get Categories: %v", err)
	}

	matcher := category.NewMatcher(categories)

	// Create test expenses
	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, storage.ChargeType, nil),
		storage.NewExpense(0, "Test Source", "Uber ride", "USD", -50000, now, storage.ChargeType, nil),
	}

	_, expenseError := s.InsertExpenses(expenses)
	if expenseError != nil {
		t.Fatalf("Failed to insert test expenses: %v", expenseError)
	}

	// Create router
	handler, router := New(s, matcher, logger)

	// Hit home to populate cache
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %v; got %v", http.StatusOK, resp.Status)
	}

	// Check reports are populated
	initialReportsKeys := router.sortedReportKeys

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
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %v; got %v", http.StatusOK, resp.Status)
	}

	ensureNoErrorInTemplateResponse(t, "import", resp.Body)

	// Hit home again t valiadte the cache has been busted and the reports have being updated
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %v; got %v", http.StatusOK, resp.Status)
	}

	// Check reports are populated
	if len(router.sortedReportKeys) <= len(initialReportsKeys) {
		t.Errorf("reports have not being repopulated after succesfull import")
	}
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
