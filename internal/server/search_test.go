package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestSearchHandler(t *testing.T) {
	logger := testutil.TestLogger(t)
	database := testutil.SetupTestDB(t, logger)

	// Create test categories
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery", Type: db.ExpenseCategoryType},
	}

	for _, c := range categories {
		_, err := db.CreateCategory(database, c.Name, c.Pattern, c.Type)
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	matcher := category.NewMatcher(categories)

	// Create test expenses
	expenses := []*db.Expense{
		{
			Source:      "Test Source",
			Date:        time.Now(),
			Description: "Restaurant bill",
			Amount:      -123456,
			Type:        db.ChargeType,
			Currency:    "USD",
			CategoryID:  sql.NullInt64{Int64: int64(1), Valid: true},
		},
	}

	_, err := db.InsertExpenses(database, expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	// Create server
	handler, _ := New(database, matcher, logger)

	// Create test request
	body := strings.NewReader("keyword=restaurant")
	req := httptest.NewRequest(http.MethodPost, "/search", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Serve request
	handler.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}
}
