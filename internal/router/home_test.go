package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestHomeHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	// Create test categories
	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
		storage.NewCategory(2, "Transport", "uber|taxi|transit"),
	}

	for _, c := range categories {
		_, err := s.CreateCategory(c.Name(), c.Pattern())
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	matcher := matcher.New(categories)

	// Create test expenses
	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, storage.ChargeType, nil),
		storage.NewExpense(0, "Test Source", "Uber ride", "USD", -50000, now, storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	// Create router
	handler, _ := New(s, matcher, logger)

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

			ensureNoErrorInTemplateResponse(t, fmt.Sprintf("reports: %s", tt.name), resp.Body)
		})
	}
}
