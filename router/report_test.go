package router

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestHomeHandlerRangeReport(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, now, domain.ChargeType, nil),
		domain.NewExpense(0, "Test Source", "Salary", "USD", 500000, now, domain.IncomeType, nil),
		// Data starts in January last year, so ranges from then on aren't clamped.
		domain.NewExpense(0, "Test Source", "Old bill", "USD", -1000,
			time.Date(now.Year()-1, time.January, 15, 12, 0, 0, 0, time.Local), domain.ChargeType, nil),
	}
	if _, err := s.InsertExpenses(context.Background(), user.ID(), expenses); err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler := New(s, logger)
	thisMonth := now.Format("2006-01")
	lastYear := now.Year() - 1

	tests := []struct {
		name         string
		url          string
		wantContains []string
	}{
		{
			name:         "default is current year to date",
			url:          "/",
			wantContains: []string{"Months", `class="year-tab active"`, fmt.Sprintf("%d-01", now.Year())},
		},
		{
			name:         "explicit range",
			url:          fmt.Sprintf("/?from=%d-01&to=%s", lastYear, thisMonth),
			wantContains: []string{"Months", "Per month:"},
		},
		{
			name:         "past year tab",
			url:          fmt.Sprintf("/?from=%d-01&to=%d-12", lastYear, lastYear),
			wantContains: []string{fmt.Sprintf("%d Summary", lastYear), "No transactions"},
		},
		{
			name:         "reversed range is swapped",
			url:          fmt.Sprintf("/?from=%s&to=%d-01", thisMonth, now.Year()),
			wantContains: []string{"Months"},
		},
		{
			name:         "invalid params fall back to default",
			url:          "/?from=nope&to=2024-13",
			wantContains: []string{"Months"},
		},
		{
			name: "open_category redirect keeps the range",
			url:  fmt.Sprintf("/?from=%d-01&to=%s&open_category=uncategorized+charge", now.Year(), thisMonth),
			wantContains: []string{
				`class="category-item" open`,
				fmt.Sprintf("redirect_to=%%2f%%3ffrom%%3d%d-01", now.Year()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status OK; got %v", w.Code)
			}
			body := w.Body.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("response missing %q", want)
				}
			}
			ensureNoErrorInTemplateResponse(t, "reports: "+tt.name, io.NopCloser(strings.NewReader(body)))
		})
	}
}

func TestBuildPeriodSelector(t *testing.T) {
	now := time.Date(2026, time.November, 9, 0, 0, 0, 0, time.Local)
	month := func(year int, m time.Month) time.Time {
		return time.Date(year, m, 1, 0, 0, 0, 0, time.Local)
	}

	selector := buildPeriodSelector([]int{2026, 2025}, month(2025, time.January), month(2025, time.December), now)

	if len(selector.Years) != 2 {
		t.Fatalf("len(Years) = %d, want 2", len(selector.Years))
	}

	current, past := selector.Years[0], selector.Years[1]
	if current.URL != "/?from=2026-01&to=2026-11" {
		t.Errorf("current year URL = %q, want year-to-date", current.URL)
	}
	if current.Active {
		t.Error("current year tab should not be active")
	}
	if past.URL != "/?from=2025-01&to=2025-12" || !past.Active {
		t.Errorf("past year tab = %+v, want active full-year link", past)
	}
	if selector.Max != "2026-11" {
		t.Errorf("Max = %q, want 2026-11", selector.Max)
	}
}

func TestHomeHandlerChartFollowsSelectedRange(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	lastYear := now.Year() - 1
	expenses := []domain.Expense{
		domain.NewExpense(0, "src", "this year", "USD", -1000, now, domain.ChargeType, nil),
		domain.NewExpense(0, "src", "last year", "USD", -2000,
			time.Date(lastYear, time.March, 10, 12, 0, 0, 0, time.Local), domain.ChargeType, nil),
	}
	if _, err := s.InsertExpenses(context.Background(), user.ID(), expenses); err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	handler := New(s, logger)

	tests := []struct {
		name        string
		url         string
		wantChart   []string
		unwantChart []string
	}{
		{
			// Data starts in March, so February is clamped off the front.
			name:      "range limits chart months and starts at first transaction",
			url:       fmt.Sprintf("/?from=%d-02&to=%d-04", lastYear, lastYear),
			wantChart: []string{fmt.Sprintf("March %d", lastYear), fmt.Sprintf("April %d", lastYear)},
			unwantChart: []string{
				fmt.Sprintf("January %d", lastYear),
				fmt.Sprintf("February %d", lastYear),
				fmt.Sprintf("May %d", lastYear),
			},
		},
		{
			name:        "open_month shows that month's year from first transaction",
			url:         fmt.Sprintf("/?open_month=3&open_year=%d", lastYear),
			wantChart:   []string{fmt.Sprintf("March %d", lastYear), fmt.Sprintf("December %d", lastYear)},
			unwantChart: []string{fmt.Sprintf("February %d", lastYear), fmt.Sprintf("January %d", now.Year())},
		},
		{
			// Future months have no data yet, so the range ends this month.
			name:        "future months are clamped off the end",
			url:         fmt.Sprintf("/?from=%d-01&to=%d-06", now.Year(), now.Year()+1),
			wantChart:   []string{fmt.Sprintf("%s %d", now.Month(), now.Year())},
			unwantChart: []string{strconv.Itoa(now.Year() + 1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status OK; got %v", w.Code)
			}
			body := w.Body.String()

			// Only inspect the chart's data attribute, not the rest of the page.
			start := strings.Index(body, `data-finance="`)
			if start == -1 {
				t.Fatal("chart canvas not rendered")
			}
			end := strings.Index(body[start+len(`data-finance="`):], `"`)
			chart := body[start : start+len(`data-finance="`)+end]

			for _, want := range tt.wantChart {
				if !strings.Contains(chart, want) {
					t.Errorf("chart data missing %q", want)
				}
			}
			for _, unwant := range tt.unwantChart {
				if strings.Contains(chart, unwant) {
					t.Errorf("chart data should not contain %q", unwant)
				}
			}
			if strings.Contains(body, "prev-period") || strings.Contains(body, "current-period") {
				t.Error("chart navigation controls should be removed")
			}
		})
	}
}

func TestHomeHandlerShareToggle(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		expenses   []domain.Expense
		wantToggle bool
	}{
		{
			name: "shown when the period has income",
			expenses: []domain.Expense{
				domain.NewExpense(0, "src", "salary", "USD", 500000, now, domain.IncomeType, nil),
				domain.NewExpense(0, "src", "rent", "USD", -100000, now, domain.ChargeType, nil),
			},
			wantToggle: true,
		},
		{
			name: "hidden without income",
			expenses: []domain.Expense{
				domain.NewExpense(0, "src", "rent", "USD", -100000, now, domain.ChargeType, nil),
			},
			wantToggle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.TestLogger(t)
			s, user := testutil.SetupTestStorage(t, logger)
			if _, err := s.InsertExpenses(context.Background(), user.ID(), tt.expenses); err != nil {
				t.Fatalf("Failed to insert test expenses: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			testutil.SetupAuthCookie(t, s, req, user, sessionCookieName, sessionDuration)
			w := httptest.NewRecorder()

			New(s, logger).ServeHTTP(w, req)

			body := w.Body.String()
			hasToggle := strings.Contains(body, `class="share-toggle"`)
			if hasToggle != tt.wantToggle {
				t.Errorf("share toggle present = %v, want %v", hasToggle, tt.wantToggle)
			}
			if tt.wantToggle && !strings.Contains(body, `class="share-of-income" title="Share of total income">20.0%`) {
				t.Error("expected rent at 20.0% of income")
			}
		})
	}
}
