package router

import (
	"bytes"
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
	_, err := db.CreateCategory(database, "Food", "restaurant|food|grocery")
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	_, err = db.CreateCategory(database, "Transport", "uber|taxi|transit")
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
			CategoryID:  intPtr(1),
		},
		{
			Source:      "Test Source",
			Date:        now,
			Description: "Uber ride",
			Amount:      -50000,
			Type:        db.ChargeType,
			Currency:    "USD",
			CategoryID:  intPtr(2),
		},
	}

	dbErrors := db.InsertExpenses(database, expenses)
	if len(dbErrors) > 0 {
		t.Fatalf("Failed to insert test expenses: %v", dbErrors)
	}

	// Create router
	handler, router := New(database, matcher)

	// Hit home to populate cache
	req := httptest.NewRequest("GET", "/", nil)
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
		t.Errorf(err.Error())
	}

	// copy file content into multipart section dataPart
	f, err := os.Open("test_data/import.json")
	if err != nil {
		t.Errorf(err.Error())
	}
	_, err = io.Copy(dataPart, f)
	if err != nil {
		t.Errorf(err.Error())
	}

	err = writer.Close()

	if err != nil {
		t.Errorf(err.Error())
	}

	req = httptest.NewRequest("POST", "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %v; got %v", http.StatusOK, resp.Status)
	}

	// Hit home again t valiadte the cache has been busted and the reports have being updated
	req = httptest.NewRequest("GET", "/", nil)
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
