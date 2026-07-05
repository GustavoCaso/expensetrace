package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestHomeHandlerWithOpenParams(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, storage.ChargeType, nil),
	}
	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler, _ := New(s, logger)

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "full page with open_month and open_year",
			url:  fmt.Sprintf("/?open_month=%d&open_year=%d", int(now.Month()), now.Year()),
		},
		{
			name: "full page with open_month, open_year, and open_category",
			url:  fmt.Sprintf("/?open_month=%d&open_year=%d&open_category=Food", int(now.Month()), now.Year()),
		},
		{
			name: "invalid open_month falls back gracefully",
			url:  "/?open_month=invalid&open_year=2024",
		},
		{
			name: "invalid open_year falls back gracefully",
			url:  "/?open_month=1&open_year=invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK; got %v", resp.Status)
			}
			ensureNoErrorInTemplateResponse(t, fmt.Sprintf("reports: %s", tt.name), resp.Body)
		})
	}
}

func TestHomeHandlerHTMXPartialSwap(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, storage.ChargeType, nil),
	}
	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler, _ := New(s, logger)

	// ?month=X&year=Y triggers HTMX partial (no full page layout)
	url := fmt.Sprintf("/?month=%d&year=%d", int(now.Month()), now.Year())
	req := httptest.NewRequest(http.MethodGet, url, nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	// Partial response should contain report card content but not full page layout
	body := w.Body.String()
	if !strings.Contains(body, "Summary") {
		t.Error("Partial response should contain report card with 'Summary'")
	}
}

func TestHomeHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	// Create test expenses
	now := time.Now()
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, storage.ChargeType, nil),
		storage.NewExpense(0, "Test Source", "Uber ride", "USD", -50000, now, storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	// Create router
	handler, _ := New(s, logger)

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
			testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
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
