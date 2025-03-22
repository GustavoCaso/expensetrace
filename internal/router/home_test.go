package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/report"
)

func TestHomeHandler(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
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
			CategoryID:  1,
		},
		{
			Source:      "Test Source",
			Date:        now,
			Description: "Uber ride",
			Amount:      -50000,
			Type:        db.ChargeType,
			Currency:    "USD",
			CategoryID:  2,
		},
	}

	err := db.InsertExpenses(database, expenses)
	if len(err) > 0 {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	// Create router
	handler := New(database, matcher)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "Default home page",
			url:            "/",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Home page with month and year",
			url:            "/?month=1&year=2024",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Home page with invalid month",
			url:            "/?month=invalid&year=2024",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Home page with invalid year",
			url:            "/?month=1&year=invalid",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %v; got %v", tt.expectedStatus, resp.Status)
			}
		})
	}
}

func TestGenerateLinks(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
	}
	matcher := category.NewMatcher(categories)

	// Create router
	r := newTestRouter(database, matcher)

	// Test with empty reports
	links := r.generateLinks()
	if len(links) != 0 {
		t.Errorf("Expected 0 links, got %d", len(links))
	}

	// Add a test report
	now := time.Now()
	reportKey := now.Format("2006-1")
	r.reports = map[string]report.Report{
		reportKey: {
			Income:   1000,
			Spending: 500,
			Savings:  500,
		},
	}
	r.sortedReportKeys = []string{reportKey}

	links = r.generateLinks()
	if len(links) != 1 {
		t.Errorf("Expected 1 link, got %d", len(links))
	}

	if links[0].Income != 1000 || links[0].Spending != 500 || links[0].Savings != 500 {
		t.Error("Link data does not match report data")
	}
}
