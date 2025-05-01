package router

import (
	"bytes"
	"database/sql"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestImport(t *testing.T) {
	database := testutil.SetupTestDB(t)

	// Create test categories
	_, err := db.CreateCategory(database, "Food", "restaurant|food|grocery", db.ExpenseCategoryType)
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	_, err = db.CreateCategory(database, "Transport", "uber|taxi|transit", db.ExpenseCategoryType)
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	categories, err := db.GetCategories(database)
	if err != nil {
		t.Fatalf("Failed to get Categories: %v", err)
	}

	matcher := category.NewMatcher(categories)

	// Create test expenses
	now := time.Now()
	expenses := []*db.Expense{
		{
			Source:      "Test Source",
			Date:        now,
			Description: "Restaurant bill",
			Amount:      -123456,
			Type:        db.ChargeType,
			Currency:    "USD",
			CategoryID:  sql.NullInt64{Int64: int64(1), Valid: true},
		},
		{
			Source:      "Test Source",
			Date:        now,
			Description: "Uber ride",
			Amount:      -50000,
			Type:        db.ChargeType,
			Currency:    "USD",
			CategoryID:  sql.NullInt64{Int64: int64(2), Valid: true},
		},
	}

	_, expenseError := db.InsertExpenses(database, expenses)
	if expenseError != nil {
		t.Fatalf("Failed to insert test expenses: %v", expenseError)
	}

	// Create router
	handler, router := New(database, matcher)

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
