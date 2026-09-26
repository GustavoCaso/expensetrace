package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/testutil"
)

func TestHomeHandlerWithOpenParams(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, domain.ChargeType, nil),
	}
	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler := New(s, logger)

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
	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, domain.ChargeType, nil),
	}
	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler := New(s, logger)

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
	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, domain.ChargeType, nil),
		domain.NewExpense(0, "Test Source", "Uber ride", "USD", -50000, now, domain.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	// Create router
	handler := New(s, logger)

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

func TestHomeHandlerCategoryBreakdown(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	ctx := context.Background()

	categoryID, err := s.CreateCategory(ctx, user.ID(), "Food & Drinks", "restaurant", 10000)
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	now := time.Now()
	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "Restaurant bill", "USD", -5000, now, domain.ChargeType, &categoryID),
		domain.NewExpense(0, "Test Source", "Rent", "USD", -123456, now, domain.ChargeType, nil),
	}
	if _, err = s.InsertExpenses(ctx, user.ID(), expenses); err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler := New(s, logger)

	url := fmt.Sprintf(
		"/?month=%d&year=%d&open_category=Food+%%26+Drinks",
		int(now.Month()),
		now.Year(),
	)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status OK; got %v", w.Code)
	}

	body := w.Body.String()

	uncategorized := strings.Index(body, "uncategorized charge")
	food := strings.Index(body, "Food &amp; Drinks")
	if uncategorized == -1 || food == -1 {
		t.Fatalf("Expected both categories in breakdown; got:\n%s", body)
	}
	if uncategorized > food {
		t.Error("Expected categories ordered by amount, largest first")
	}

	if strings.Count(body, `<details class="category-item" open>`) != 1 {
		t.Error("Expected exactly one category to be open")
	}
	openAt := strings.Index(body, `<details class="category-item" open>`)
	if openAt > food {
		t.Error("Expected open_category to be the open category")
	}

	redirect := "redirect_to=%2f%3fopen_category%3dFood%2b%2526%2bDrinks%26open_month%3d"
	if !strings.Contains(body, redirect) {
		t.Errorf("Expected expense link with encoded redirect %q", redirect)
	}
}
