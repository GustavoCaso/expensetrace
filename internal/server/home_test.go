package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestHomeHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery", Type: db.ExpenseCategoryType},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit", Type: db.ExpenseCategoryType},
	}

	for _, c := range categories {
		_, err := db.CreateCategory(database, c.Name, c.Pattern, c.Type)
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
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

	_, err := db.InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	// Create server
	handler, _ := New(database, matcher, logger)

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
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %v; got %v", tt.expectedStatus, resp.Status)
			}
		})
	}
}
