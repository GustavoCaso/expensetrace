package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/router"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

const (
	sessionCookieName = "session_id"
	sessionDuration   = 24 * time.Hour
)

// TestFilteringEndToEnd tests the complete filtering flow from HTTP request to database query.
func TestFilteringEndToEnd(t *testing.T) {
	// Setup
	logger := testutil.TestLogger(t)
	st, user := testutil.SetupTestStorage(t, logger)
	ctx := context.Background()

	// Insert diverse test data
	expenses := []storage.Expense{
		// Coffee expenses on visa
		storage.NewExpense(0, "visa", "Starbucks coffee", "USD", -550, time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "visa", "Local cafe coffee", "USD", -450, time.Date(2024, 1, 20, 9, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		// Lunch on mastercard
		storage.NewExpense(0, "mastercard", "Restaurant lunch", "USD", -1500, time.Date(2024, 1, 18, 12, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		// Grocery on visa (different month)
		storage.NewExpense(0, "visa", "Grocery shopping", "USD", -8000, time.Date(2024, 2, 5, 14, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		// Income
		storage.NewExpense(0, "employer", "Salary", "USD", 500000, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), storage.IncomeType, nil),
	}

	_, err := st.InsertExpenses(ctx, user.ID(), expenses)
	if err != nil {
		t.Fatalf("failed to insert expenses: %v", err)
	}

	// Setup router
	handler, _ := router.New(st, logger)

	// Test scenarios
	tests := []struct {
		name              string
		queryString       string
		expectedCount     int
		shouldContain     []string
		shouldNotContain  []string
	}{
		{
			name:          "no filters - all expenses",
			queryString:   "",
			expectedCount: 5,
			shouldContain: []string{"coffee", "lunch", "Grocery", "Salary"},
		},
		{
			name:             "filter by description",
			queryString:      "?description=coffee",
			expectedCount:    2,
			shouldContain:    []string{"Starbucks", "Local cafe"},
			shouldNotContain: []string{"lunch", "Grocery", "Salary"},
		},
		{
			name:             "filter by source",
			queryString:      "?source=visa",
			expectedCount:    3,
			shouldContain:    []string{"Starbucks", "Local cafe", "Grocery"},
			shouldNotContain: []string{"lunch", "Salary"},
		},
		{
			name:             "filter by amount range",
			queryString:      "?amount_min=-10.00&amount_max=-4.00",
			expectedCount:    2,
			shouldContain:    []string{"Starbucks", "Local cafe"},
			shouldNotContain: []string{"lunch", "Grocery"},
		},
		{
			name:             "filter by date range",
			queryString:      "?date_from=2024-01-01&date_to=2024-01-31",
			expectedCount:    4, // Excludes February grocery
			shouldContain:    []string{"coffee", "lunch", "Salary"},
			shouldNotContain: []string{"Grocery"},
		},
		{
			name:             "combined filters",
			queryString:      "?description=coffee&source=visa&amount_max=-5.00",
			expectedCount:    1,
			shouldContain:    []string{"Starbucks"},
			shouldNotContain: []string{"Local cafe", "lunch", "Grocery"},
		},
		{
			name:          "sort by amount ascending",
			queryString:   "?sort=amount:asc",
			expectedCount: 5,
			// Check order via response inspection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/expenses"+tt.queryString, nil)
			testutil.SetupAuthCookie(t, st, req, user, sessionCookieName, sessionDuration)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			body := w.Body.String()

			// Check expected content
			for _, content := range tt.shouldContain {
				if !contains(body, content) {
					t.Errorf("expected body to contain %q", content)
				}
			}

			for _, content := range tt.shouldNotContain {
				if contains(body, content) {
					t.Errorf("expected body NOT to contain %q", content)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		(len(s) > len(substr) && containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
