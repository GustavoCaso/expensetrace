# Expense Filtering and Sorting Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a flexible filtering and sorting system for the expenses view with a persistent filter bar.

**Architecture:** Specification Pattern with type-safe filter objects. Four layers: Filter specs (structs), URL parsing (query params → structs), Storage (SQL generation), Handlers (integration).

**Tech Stack:** Go 1.23+, SQLite, Go templates, native HTTP

---

## Task 1: Create Filter Package Structure

**Files:**
- Create: `internal/filter/filter.go`
- Create: `internal/filter/filter_test.go`

**Step 1: Create filter package directory**

Run: `mkdir -p internal/filter`

**Step 2: Write filter specification structs**

Create `internal/filter/filter.go`:

```go
package filter

import "time"

// ExpenseFilter holds filter criteria for expense queries.
// All fields are pointers to distinguish "not set" from zero values.
type ExpenseFilter struct {
	Description *string    // LIKE search on description
	Source      *string    // LIKE search on source
	AmountMin   *int64     // Minimum amount in cents (inclusive)
	AmountMax   *int64     // Maximum amount in cents (inclusive)
	DateFrom    *time.Time // Start date (inclusive)
	DateTo      *time.Time // End date (inclusive)
}

// SortField represents a field that can be sorted on.
type SortField string

const (
	SortByDate   SortField = "date"
	SortByAmount SortField = "amount"
)

// SortDirection represents sort order.
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// SortOptions holds sorting preferences.
type SortOptions struct {
	Field     SortField
	Direction SortDirection
}

// DefaultSortOptions returns the default sort (date descending, newest first).
func DefaultSortOptions() *SortOptions {
	return &SortOptions{
		Field:     SortByDate,
		Direction: SortDesc,
	}
}

// String returns the sort options as a string (e.g., "date:desc").
func (s *SortOptions) String() string {
	return string(s.Field) + ":" + string(s.Direction)
}
```

**Step 3: Write tests for default sort options**

Create `internal/filter/filter_test.go`:

```go
package filter

import "testing"

func TestDefaultSortOptions(t *testing.T) {
	opts := DefaultSortOptions()

	if opts.Field != SortByDate {
		t.Errorf("expected field %q, got %q", SortByDate, opts.Field)
	}

	if opts.Direction != SortDesc {
		t.Errorf("expected direction %q, got %q", SortDesc, opts.Direction)
	}
}

func TestSortOptionsString(t *testing.T) {
	tests := []struct {
		name     string
		opts     *SortOptions
		expected string
	}{
		{
			name:     "date descending",
			opts:     &SortOptions{Field: SortByDate, Direction: SortDesc},
			expected: "date:desc",
		},
		{
			name:     "date ascending",
			opts:     &SortOptions{Field: SortByDate, Direction: SortAsc},
			expected: "date:asc",
		},
		{
			name:     "amount descending",
			opts:     &SortOptions{Field: SortByAmount, Direction: SortDesc},
			expected: "amount:desc",
		},
		{
			name:     "amount ascending",
			opts:     &SortOptions{Field: SortByAmount, Direction: SortAsc},
			expected: "amount:asc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.opts.String()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/filter/...`
Expected: PASS

---

## Task 2: Implement URL Parser - Amount Parsing

**Files:**
- Create: `internal/filter/parser.go`
- Modify: `internal/filter/filter_test.go`

**Step 1: Write test for parseAmount helper**

Add to `internal/filter/filter_test.go`:

```go
func TestParseAmount(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
		wantErr  bool
	}{
		{
			name:     "valid whole number",
			input:    "10",
			expected: 1000,
			wantErr:  false,
		},
		{
			name:     "valid decimal",
			input:    "10.50",
			expected: 1050,
			wantErr:  false,
		},
		{
			name:     "zero",
			input:    "0",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "many decimals rounds",
			input:    "10.999",
			expected: 1099,
			wantErr:  false,
		},
		{
			name:    "invalid format",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAmount(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/filter/... -v`
Expected: FAIL with "undefined: parseAmount"

**Step 3: Implement parseAmount**

Create `internal/filter/parser.go`:

```go
package filter

import (
	"fmt"
	"strconv"
)

// parseAmount converts a dollar amount string to cents.
// Examples: "10.50" -> 1050, "5" -> 500
func parseAmount(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("amount cannot be empty")
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format: %w", err)
	}

	cents := int64(f * 100)
	return cents, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/filter/... -v`
Expected: PASS

---

## Task 3: Implement URL Parser - Sort Parsing

**Files:**
- Modify: `internal/filter/parser.go`
- Modify: `internal/filter/filter_test.go`

**Step 1: Write test for parseSort helper**

Add to `internal/filter/filter_test.go`:

```go
func TestParseSort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *SortOptions
		wantErr  bool
	}{
		{
			name:  "date descending",
			input: "date:desc",
			expected: &SortOptions{
				Field:     SortByDate,
				Direction: SortDesc,
			},
			wantErr: false,
		},
		{
			name:  "date ascending",
			input: "date:asc",
			expected: &SortOptions{
				Field:     SortByDate,
				Direction: SortAsc,
			},
			wantErr: false,
		},
		{
			name:  "amount descending",
			input: "amount:desc",
			expected: &SortOptions{
				Field:     SortByAmount,
				Direction: SortDesc,
			},
			wantErr: false,
		},
		{
			name:  "amount ascending",
			input: "amount:asc",
			expected: &SortOptions{
				Field:     SortByAmount,
				Direction: SortAsc,
			},
			wantErr: false,
		},
		{
			name:    "invalid field",
			input:   "invalid:desc",
			wantErr: true,
		},
		{
			name:    "invalid direction",
			input:   "date:invalid",
			wantErr: true,
		},
		{
			name:    "missing colon",
			input:   "date",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSort(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result.Field != tt.expected.Field {
				t.Errorf("expected field %q, got %q", tt.expected.Field, result.Field)
			}

			if result.Direction != tt.expected.Direction {
				t.Errorf("expected direction %q, got %q", tt.expected.Direction, result.Direction)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/filter/... -v -run TestParseSort`
Expected: FAIL with "undefined: parseSort"

**Step 3: Implement parseSort**

Add to `internal/filter/parser.go`:

```go
import (
	"fmt"
	"strconv"
	"strings"
)

// parseSort parses a sort string like "date:desc" into SortOptions.
func parseSort(s string) (*SortOptions, error) {
	if s == "" {
		return nil, fmt.Errorf("sort string cannot be empty")
	}

	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid sort format, expected field:direction")
	}

	field := SortField(parts[0])
	direction := SortDirection(parts[1])

	// Validate field
	if field != SortByDate && field != SortByAmount {
		return nil, fmt.Errorf("invalid sort field: %s (must be date or amount)", field)
	}

	// Validate direction
	if direction != SortAsc && direction != SortDesc {
		return nil, fmt.Errorf("invalid sort direction: %s (must be asc or desc)", direction)
	}

	return &SortOptions{
		Field:     field,
		Direction: direction,
	}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/filter/... -v`
Expected: PASS

---

## Task 4: Implement Main URL Parser

**Files:**
- Modify: `internal/filter/parser.go`
- Modify: `internal/filter/filter_test.go`

**Step 1: Write test for ParseExpenseFilters**

Add to `internal/filter/filter_test.go`:

```go
import (
	"net/url"
	"time"
)

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestParseExpenseFilters(t *testing.T) {
	tests := []struct {
		name         string
		queryString  string
		wantFilter   *ExpenseFilter
		wantSort     *SortOptions
		wantErr      bool
	}{
		{
			name:        "no filters - returns defaults",
			queryString: "",
			wantFilter:  &ExpenseFilter{},
			wantSort:    DefaultSortOptions(),
			wantErr:     false,
		},
		{
			name:        "description filter only",
			queryString: "description=coffee",
			wantFilter: &ExpenseFilter{
				Description: stringPtr("coffee"),
			},
			wantSort: DefaultSortOptions(),
			wantErr:  false,
		},
		{
			name:        "source filter only",
			queryString: "source=visa",
			wantFilter: &ExpenseFilter{
				Source: stringPtr("visa"),
			},
			wantSort: DefaultSortOptions(),
			wantErr:  false,
		},
		{
			name:        "amount min filter",
			queryString: "amount_min=10.50",
			wantFilter: &ExpenseFilter{
				AmountMin: int64Ptr(1050),
			},
			wantSort: DefaultSortOptions(),
			wantErr:  false,
		},
		{
			name:        "amount max filter",
			queryString: "amount_max=20.00",
			wantFilter: &ExpenseFilter{
				AmountMax: int64Ptr(2000),
			},
			wantSort: DefaultSortOptions(),
			wantErr:  false,
		},
		{
			name:        "date from filter",
			queryString: "date_from=2024-01-01",
			wantFilter: &ExpenseFilter{
				DateFrom: timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
			wantSort: DefaultSortOptions(),
			wantErr:  false,
		},
		{
			name:        "date to filter",
			queryString: "date_to=2024-01-31",
			wantFilter: &ExpenseFilter{
				DateTo: timePtr(time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)),
			},
			wantSort: DefaultSortOptions(),
			wantErr:  false,
		},
		{
			name:        "custom sort",
			queryString: "sort=amount:asc",
			wantFilter:  &ExpenseFilter{},
			wantSort: &SortOptions{
				Field:     SortByAmount,
				Direction: SortAsc,
			},
			wantErr: false,
		},
		{
			name:        "all filters combined",
			queryString: "description=coffee&source=visa&amount_min=5.00&amount_max=10.00&date_from=2024-01-01&date_to=2024-01-31&sort=amount:desc",
			wantFilter: &ExpenseFilter{
				Description: stringPtr("coffee"),
				Source:      stringPtr("visa"),
				AmountMin:   int64Ptr(500),
				AmountMax:   int64Ptr(1000),
				DateFrom:    timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				DateTo:      timePtr(time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)),
			},
			wantSort: &SortOptions{
				Field:     SortByAmount,
				Direction: SortDesc,
			},
			wantErr: false,
		},
		{
			name:        "invalid amount min",
			queryString: "amount_min=invalid",
			wantErr:     true,
		},
		{
			name:        "invalid date from",
			queryString: "date_from=not-a-date",
			wantErr:     true,
		},
		{
			name:        "invalid sort",
			queryString: "sort=invalid:format",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, _ := url.ParseQuery(tt.queryString)
			filter, sort, err := ParseExpenseFilters(params)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Compare filter fields
			if !equalStringPtr(filter.Description, tt.wantFilter.Description) {
				t.Errorf("Description: expected %v, got %v", tt.wantFilter.Description, filter.Description)
			}
			if !equalStringPtr(filter.Source, tt.wantFilter.Source) {
				t.Errorf("Source: expected %v, got %v", tt.wantFilter.Source, filter.Source)
			}
			if !equalInt64Ptr(filter.AmountMin, tt.wantFilter.AmountMin) {
				t.Errorf("AmountMin: expected %v, got %v", tt.wantFilter.AmountMin, filter.AmountMin)
			}
			if !equalInt64Ptr(filter.AmountMax, tt.wantFilter.AmountMax) {
				t.Errorf("AmountMax: expected %v, got %v", tt.wantFilter.AmountMax, filter.AmountMax)
			}
			if !equalTimePtr(filter.DateFrom, tt.wantFilter.DateFrom) {
				t.Errorf("DateFrom: expected %v, got %v", tt.wantFilter.DateFrom, filter.DateFrom)
			}
			if !equalTimePtr(filter.DateTo, tt.wantFilter.DateTo) {
				t.Errorf("DateTo: expected %v, got %v", tt.wantFilter.DateTo, filter.DateTo)
			}

			// Compare sort
			if sort.Field != tt.wantSort.Field {
				t.Errorf("Sort field: expected %q, got %q", tt.wantSort.Field, sort.Field)
			}
			if sort.Direction != tt.wantSort.Direction {
				t.Errorf("Sort direction: expected %q, got %q", tt.wantSort.Direction, sort.Direction)
			}
		})
	}
}

func equalStringPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalInt64Ptr(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalTimePtr(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/filter/... -v -run TestParseExpenseFilters`
Expected: FAIL with "undefined: ParseExpenseFilters"

**Step 3: Implement ParseExpenseFilters**

Add to `internal/filter/parser.go`:

```go
import (
	"net/url"
	"time"
)

// ParseExpenseFilters parses URL query parameters into filter and sort options.
func ParseExpenseFilters(params url.Values) (*ExpenseFilter, *SortOptions, error) {
	filter := &ExpenseFilter{}
	sort := DefaultSortOptions()

	// Parse string filters
	if desc := params.Get("description"); desc != "" {
		filter.Description = &desc
	}

	if src := params.Get("source"); src != "" {
		filter.Source = &src
	}

	// Parse amount range
	if minStr := params.Get("amount_min"); minStr != "" {
		val, err := parseAmount(minStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid amount_min: %w", err)
		}
		filter.AmountMin = &val
	}

	if maxStr := params.Get("amount_max"); maxStr != "" {
		val, err := parseAmount(maxStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid amount_max: %w", err)
		}
		filter.AmountMax = &val
	}

	// Parse date range
	if fromStr := params.Get("date_from"); fromStr != "" {
		val, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid date_from: %w", err)
		}
		filter.DateFrom = &val
	}

	if toStr := params.Get("date_to"); toStr != "" {
		val, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid date_to: %w", err)
		}
		filter.DateTo = &val
	}

	// Parse sort
	if sortStr := params.Get("sort"); sortStr != "" {
		parsed, err := parseSort(sortStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid sort: %w", err)
		}
		sort = parsed
	}

	return filter, sort, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/filter/... -v`
Expected: PASS

---

## Task 5: Add Storage Interface Method

**Files:**
- Modify: `internal/storage/storage.go`

**Step 1: Add GetExpensesFiltered to Storage interface**

Add to `internal/storage/storage.go` after the existing expense methods (around line 266):

```go
import "github.com/GustavoCaso/expensetrace/internal/filter"

// In the Storage interface, add after GetExpensesByCategory:
GetExpensesFiltered(ctx context.Context, userID int64, expFilter *filter.ExpenseFilter, sort *filter.SortOptions) ([]Expense, error)
```

**Step 2: Verify it compiles (will fail - no implementation yet)**

Run: `go build ./...`
Expected: FAIL with "does not implement Storage (missing GetExpensesFiltered method)"


---

## Task 6: Implement Storage Layer - Basic Query

**Files:**
- Modify: `internal/storage/sqlite/expenses.go`
- Create: `internal/storage/sqlite/expenses_filtered_test.go`

**Step 1: Write test for GetExpensesFiltered with no filters**

Create `internal/storage/sqlite/expenses_filtered_test.go`:

```go
package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/filter"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestGetExpensesFiltered_NoFilters(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	ctx := context.Background()
	userID := int64(1)

	// Insert test expenses
	expenses := []storage.Expense{
		storage.NewExpense(0, "store1", "coffee", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store2", "lunch", "USD", -1200, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "employer", "salary", "USD", 500000, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), storage.IncomeType, nil),
	}

	_, err := s.InsertExpenses(ctx, userID, expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	// Query with no filters
	emptyFilter := &filter.ExpenseFilter{}
	sortOptions := filter.DefaultSortOptions()

	results, err := s.GetExpensesFiltered(ctx, userID, emptyFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Verify default sort (date desc) - salary should be last
	if results[0].Description() != "lunch" {
		t.Errorf("expected first result to be 'lunch', got %q", results[0].Description())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/storage/sqlite/... -v -run TestGetExpensesFiltered_NoFilters`
Expected: FAIL with "undefined: s.GetExpensesFiltered"

**Step 3: Implement GetExpensesFiltered basic version**

Add to `internal/storage/sqlite/expenses.go`:

```go
import "github.com/GustavoCaso/expensetrace/internal/filter"

func (s *sqliteStorage) GetExpensesFiltered(
	ctx context.Context,
	userID int64,
	expFilter *filter.ExpenseFilter,
	sort *filter.SortOptions,
) ([]storage.Expense, error) {
	query := "SELECT * FROM expenses WHERE user_id = ?"
	args := []interface{}{userID}

	// Add filters dynamically
	if expFilter.Description != nil {
		query += " AND description LIKE ?"
		args = append(args, "%"+*expFilter.Description+"%")
	}

	if expFilter.Source != nil {
		query += " AND source LIKE ?"
		args = append(args, "%"+*expFilter.Source+"%")
	}

	if expFilter.AmountMin != nil {
		query += " AND amount >= ?"
		args = append(args, *expFilter.AmountMin)
	}

	if expFilter.AmountMax != nil {
		query += " AND amount <= ?"
		args = append(args, *expFilter.AmountMax)
	}

	if expFilter.DateFrom != nil {
		query += " AND date >= ?"
		args = append(args, expFilter.DateFrom.Unix())
	}

	if expFilter.DateTo != nil {
		query += " AND date <= ?"
		args = append(args, expFilter.DateTo.Unix())
	}

	// Add sorting
	query += fmt.Sprintf(" ORDER BY %s %s", sort.Field, sort.Direction)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return extractExpensesFromRows(rows)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/storage/sqlite/... -v -run TestGetExpensesFiltered_NoFilters`
Expected: PASS


---

## Task 7: Test Storage Layer - Individual Filters

**Files:**
- Modify: `internal/storage/sqlite/expenses_filtered_test.go`

**Step 1: Write test for description filter**

Add to `internal/storage/sqlite/expenses_filtered_test.go`:

```go
func TestGetExpensesFiltered_DescriptionFilter(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	ctx := context.Background()
	userID := int64(1)

	// Insert test expenses
	expenses := []storage.Expense{
		storage.NewExpense(0, "store1", "morning coffee", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store2", "afternoon coffee", "USD", -450, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store3", "lunch", "USD", -1200, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(ctx, userID, expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	// Filter by description
	desc := "coffee"
	expFilter := &filter.ExpenseFilter{
		Description: &desc,
	}
	sortOptions := filter.DefaultSortOptions()

	results, err := s.GetExpensesFiltered(ctx, userID, expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Both should contain "coffee"
	for _, r := range results {
		if !contains(r.Description(), "coffee") {
			t.Errorf("expected description to contain 'coffee', got %q", r.Description())
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && len(s) > 0 && s[1:len(s)-1] != "" && contains(s[1:], substr)
}
```

Actually, let's use a simpler approach:

```go
import "strings"

func TestGetExpensesFiltered_DescriptionFilter(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	ctx := context.Background()
	userID := int64(1)

	// Insert test expenses
	expenses := []storage.Expense{
		storage.NewExpense(0, "store1", "morning coffee", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store2", "afternoon coffee", "USD", -450, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store3", "lunch", "USD", -1200, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(ctx, userID, expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	// Filter by description
	desc := "coffee"
	expFilter := &filter.ExpenseFilter{
		Description: &desc,
	}
	sortOptions := filter.DefaultSortOptions()

	results, err := s.GetExpensesFiltered(ctx, userID, expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Both should contain "coffee"
	for _, r := range results {
		if !strings.Contains(r.Description(), "coffee") {
			t.Errorf("expected description to contain 'coffee', got %q", r.Description())
		}
	}
}
```

**Step 2: Write tests for other individual filters**

Add to `internal/storage/sqlite/expenses_filtered_test.go`:

```go
func TestGetExpensesFiltered_SourceFilter(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	ctx := context.Background()
	userID := int64(1)

	expenses := []storage.Expense{
		storage.NewExpense(0, "visa", "coffee", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "mastercard", "lunch", "USD", -1200, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "visa", "dinner", "USD", -2000, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(ctx, userID, expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	source := "visa"
	expFilter := &filter.ExpenseFilter{
		Source: &source,
	}
	sortOptions := filter.DefaultSortOptions()

	results, err := s.GetExpensesFiltered(ctx, userID, expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestGetExpensesFiltered_AmountRangeFilter(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	ctx := context.Background()
	userID := int64(1)

	expenses := []storage.Expense{
		storage.NewExpense(0, "store", "small", "USD", -300, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "medium", "USD", -800, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "large", "USD", -1500, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(ctx, userID, expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	min := int64(-1000)
	max := int64(-500)
	expFilter := &filter.ExpenseFilter{
		AmountMin: &min,
		AmountMax: &max,
	}
	sortOptions := filter.DefaultSortOptions()

	results, err := s.GetExpensesFiltered(ctx, userID, expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Description() != "medium" {
		t.Errorf("expected 'medium', got %q", results[0].Description())
	}
}

func TestGetExpensesFiltered_DateRangeFilter(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	ctx := context.Background()
	userID := int64(1)

	expenses := []storage.Expense{
		storage.NewExpense(0, "store", "jan", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "feb", "USD", -600, time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "mar", "USD", -700, time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(ctx, userID, expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	from := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)
	expFilter := &filter.ExpenseFilter{
		DateFrom: &from,
		DateTo:   &to,
	}
	sortOptions := filter.DefaultSortOptions()

	results, err := s.GetExpensesFiltered(ctx, userID, expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Description() != "feb" {
		t.Errorf("expected 'feb', got %q", results[0].Description())
	}
}
```

**Step 3: Run tests to verify they pass**

Run: `go test ./internal/storage/sqlite/... -v -run TestGetExpensesFiltered`
Expected: PASS

---

## Task 8: Test Storage Layer - Combined Filters and Sorting

**Files:**
- Modify: `internal/storage/sqlite/expenses_filtered_test.go`

**Step 1: Write test for combined filters**

Add to `internal/storage/sqlite/expenses_filtered_test.go`:

```go
func TestGetExpensesFiltered_CombinedFilters(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	ctx := context.Background()
	userID := int64(1)

	expenses := []storage.Expense{
		storage.NewExpense(0, "visa", "coffee shop", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "visa", "coffee shop", "USD", -600, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "mastercard", "coffee shop", "USD", -550, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "visa", "restaurant", "USD", -1200, time.Date(2024, 1, 18, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(ctx, userID, expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	// Filter: visa + coffee + amount < -520
	source := "visa"
	desc := "coffee"
	max := int64(-520)
	expFilter := &filter.ExpenseFilter{
		Source:      &source,
		Description: &desc,
		AmountMax:   &max,
	}
	sortOptions := filter.DefaultSortOptions()

	results, err := s.GetExpensesFiltered(ctx, userID, expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if len(results) > 0 {
		if results[0].Amount() != -600 {
			t.Errorf("expected amount -600, got %d", results[0].Amount())
		}
	}
}
```

**Step 2: Write tests for sorting**

Add to `internal/storage/sqlite/expenses_filtered_test.go`:

```go
func TestGetExpensesFiltered_SortByDateAsc(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	ctx := context.Background()
	userID := int64(1)

	expenses := []storage.Expense{
		storage.NewExpense(0, "store", "third", "USD", -500, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "first", "USD", -600, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "second", "USD", -700, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(ctx, userID, expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	expFilter := &filter.ExpenseFilter{}
	sortOptions := &filter.SortOptions{
		Field:     filter.SortByDate,
		Direction: filter.SortAsc,
	}

	results, err := s.GetExpensesFiltered(ctx, userID, expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Description() != "first" {
		t.Errorf("expected 'first' at position 0, got %q", results[0].Description())
	}
	if results[1].Description() != "second" {
		t.Errorf("expected 'second' at position 1, got %q", results[1].Description())
	}
	if results[2].Description() != "third" {
		t.Errorf("expected 'third' at position 2, got %q", results[2].Description())
	}
}

func TestGetExpensesFiltered_SortByAmountDesc(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	ctx := context.Background()
	userID := int64(1)

	expenses := []storage.Expense{
		storage.NewExpense(0, "store", "medium", "USD", -800, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "small", "USD", -500, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "large", "USD", -1200, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(ctx, userID, expenses)
	if err != nil {
		t.Fatalf("failed to insert test expenses: %v", err)
	}

	expFilter := &filter.ExpenseFilter{}
	sortOptions := &filter.SortOptions{
		Field:     filter.SortByAmount,
		Direction: filter.SortDesc,
	}

	results, err := s.GetExpensesFiltered(ctx, userID, expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Desc order: largest (least negative) first
	if results[0].Description() != "small" {
		t.Errorf("expected 'small' at position 0, got %q", results[0].Description())
	}
	if results[1].Description() != "medium" {
		t.Errorf("expected 'medium' at position 1, got %q", results[1].Description())
	}
	if results[2].Description() != "large" {
		t.Errorf("expected 'large' at position 2, got %q", results[2].Description())
	}
}
```

**Step 3: Run tests to verify they pass**

Run: `go test ./internal/storage/sqlite/... -v -run TestGetExpensesFiltered`
Expected: PASS

---

## Task 9: Test Storage Layer - Multi-User Isolation

**Files:**
- Modify: `internal/storage/sqlite/expenses_filtered_test.go`

**Step 1: Write test for user isolation**

Add to `internal/storage/sqlite/expenses_filtered_test.go`:

```go
func TestGetExpensesFiltered_UserIsolation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := New(db)
	ctx := context.Background()

	user1ID := int64(1)
	user2ID := int64(2)

	// Insert expenses for user 1
	user1Expenses := []storage.Expense{
		storage.NewExpense(0, "store", "user1 expense1", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
		storage.NewExpense(0, "store", "user1 expense2", "USD", -600, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}
	_, err := s.InsertExpenses(ctx, user1ID, user1Expenses)
	if err != nil {
		t.Fatalf("failed to insert user1 expenses: %v", err)
	}

	// Insert expenses for user 2
	user2Expenses := []storage.Expense{
		storage.NewExpense(0, "store", "user2 expense1", "USD", -700, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), storage.ChargeType, nil),
	}
	_, err = s.InsertExpenses(ctx, user2ID, user2Expenses)
	if err != nil {
		t.Fatalf("failed to insert user2 expenses: %v", err)
	}

	// Query for user 1 - should only see user 1's expenses
	expFilter := &filter.ExpenseFilter{}
	sortOptions := filter.DefaultSortOptions()

	results, err := s.GetExpensesFiltered(ctx, user1ID, expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("user1: expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if !strings.Contains(r.Description(), "user1") {
			t.Errorf("user1 query returned user2's expense: %q", r.Description())
		}
	}

	// Query for user 2 - should only see user 2's expenses
	results, err = s.GetExpensesFiltered(ctx, user2ID, expFilter, sortOptions)
	if err != nil {
		t.Fatalf("GetExpensesFiltered failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("user2: expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && !strings.Contains(results[0].Description(), "user2") {
		t.Errorf("user2 query returned user1's expense: %q", results[0].Description())
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/storage/sqlite/... -v -run TestGetExpensesFiltered_UserIsolation`
Expected: PASS

---

## Task 10: Update Handler to Use Filtered Queries

**Files:**
- Modify: `internal/router/expense.go`

**Step 1: Update expensesHandler signature**

Modify the `expensesHandler` method in `internal/router/expense.go`:

Change from:
```go
func (c *expenseHandler) expensesHandler(ctx context.Context, w http.ResponseWriter, banner *banner) {
```

To:
```go
func (c *expenseHandler) expensesHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, banner *banner) {
```

**Step 2: Update expensesHandler implementation**

Replace the implementation of `expensesHandler`:

```go
func (c *expenseHandler) expensesHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, banner *banner) {
	userID := userIDFromContext(ctx)
	base := newViewBase(ctx, c.storage, c.logger, pageExpenses)
	data := expesesViewData{
		viewBase: base,
	}

	defer func() {
		c.templates.Render(w, "pages/expenses/index.html", data)
	}()

	// Parse filters from URL
	expenseFilter, sortOptions, err := filter.ParseExpenseFilters(r.URL.Query())
	if err != nil {
		data.Error = fmt.Sprintf("Invalid filters: %s", err.Error())
		return
	}

	// Get filtered expenses
	expenses, err := c.storage.GetExpensesFiltered(ctx, userID, expenseFilter, sortOptions)
	if err != nil {
		data.Error = err.Error()
		return
	}

	// Group by year and month (existing logic)
	groupedExpenses, years, err := expensesGroupByYearAndMonth(ctx, userID, expenses, c.storage)
	if err != nil {
		data.Error = fmt.Sprintf("Error grouping expenses: %s", err.Error())
		return
	}

	today := time.Now()
	data.Expenses = groupedExpenses
	data.Years = years
	data.Months = months
	data.CurrentYear = today.Year()
	data.CurrentMonth = today.Month().String()

	// Pass filter state to template for rendering filter bar
	data.Filter = expenseFilter
	data.Sort = sortOptions

	if banner != nil {
		data.Banner = *banner
	}
}
```

**Step 3: Update expesesViewData struct**

Modify `expesesViewData` in `internal/router/expense.go`:

```go
import "github.com/GustavoCaso/expensetrace/internal/filter"

type expesesViewData struct {
	viewBase
	Expenses     expensesByYear
	Years        []int
	Months       []string
	CurrentYear  int
	CurrentMonth string
	Filter       *filter.ExpenseFilter  // NEW
	Sort         *filter.SortOptions    // NEW
}
```

**Step 4: Update route registration**

Update the route in `RegisterRoutes` method:

Change from:
```go
mux.HandleFunc("GET /expenses", func(w http.ResponseWriter, r *http.Request) {
	c.expensesHandler(r.Context(), w, nil)
})
```

To:
```go
mux.HandleFunc("GET /expenses", func(w http.ResponseWriter, r *http.Request) {
	c.expensesHandler(r.Context(), w, r, nil)
})
```

**Step 5: Update deleteExpenseHandler call**

Find the `deleteExpenseHandler` method and update its call to `expensesHandler`:

Change from:
```go
c.expensesHandler(ctx, w, &banner{
	Icon:    "🔥",
	Message: "Expense deleted",
})
```

To:
```go
c.expensesHandler(ctx, w, r, &banner{
	Icon:    "🔥",
	Message: "Expense deleted",
})
```

**Step 6: Verify it compiles**

Run: `go build ./...`
Expected: Success (or compile errors to fix)

---

## Task 11: Remove Old Search Handler

**Files:**
- Modify: `internal/router/expense.go`

**Step 1: Remove expenseSearchHandler route**

In the `RegisterRoutes` method, remove:

```go
mux.HandleFunc("POST /expense/search", func(w http.ResponseWriter, r *http.Request) {
	c.expenseSearchHandler(r.Context(), w, r)
})
```

**Step 2: Remove expenseSearchHandler function**

Delete the entire `expenseSearchHandler` function (around lines 563-606).

**Step 3: Verify it compiles**

Run: `go build ./...`
Expected: Success

**Step 4: Run existing tests**

Run: `go test ./internal/router/... -v`
Expected: May have some failures in tests that reference old search handler - that's OK for now

---

## Task 12: Add Template Helper Functions

**Files:**
- Modify: `internal/router/templates.go`

**Step 1: Add amountToDollars template function**

Add to the `funcMap` in `internal/router/templates.go`:

```go
"amountToDollars": func(cents *int64) string {
	if cents == nil {
		return ""
	}
	dollars := float64(*cents) / 100.0
	return fmt.Sprintf("%.2f", dollars)
},
```

**Step 2: Add formatDate template function**

Add to the `funcMap`:

```go
"formatDate": func(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
},
```

**Step 3: Verify the full funcMap**

The `funcMap` should now include both new functions. Make sure to import necessary packages at the top of the file:

```go
import (
	"fmt"
	"time"
	// ... other imports
)
```

**Step 4: Verify it compiles**

Run: `go build ./...`
Expected: Success

---

## Task 13: Create Filter Bar Template

**Files:**
- Modify: `web/templates/pages/expenses/index.html`

**Step 1: Read current template structure**

Run: `cat web/templates/pages/expenses/index.html | head -20`

Note the current structure to understand where to insert the filter bar.

**Step 2: Add filter bar at the top of the page**

Add the filter bar form after the opening content div and before the existing expense list. Insert this code:

```html
<!-- Filter Bar - Always visible at top -->
<form method="GET" action="/expenses" class="filter-bar" style="margin-bottom: 2rem; padding: 1rem; border: 1px solid #ddd; border-radius: 4px; background-color: #f9f9f9;">
    <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 1rem;">
        <!-- Description filter -->
        <div>
            <label for="description" style="display: block; margin-bottom: 0.25rem; font-weight: bold;">Description</label>
            <input type="text"
                   name="description"
                   id="description"
                   placeholder="Search..."
                   style="width: 100%; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;"
                   value="{{if .Filter.Description}}{{.Filter.Description}}{{end}}">
        </div>

        <!-- Source filter -->
        <div>
            <label for="source" style="display: block; margin-bottom: 0.25rem; font-weight: bold;">Source</label>
            <input type="text"
                   name="source"
                   id="source"
                   placeholder="Search..."
                   style="width: 100%; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;"
                   value="{{if .Filter.Source}}{{.Filter.Source}}{{end}}">
        </div>

        <!-- Amount min filter -->
        <div>
            <label for="amount_min" style="display: block; margin-bottom: 0.25rem; font-weight: bold;">Amount Min</label>
            <input type="number"
                   name="amount_min"
                   id="amount_min"
                   step="0.01"
                   placeholder="0.00"
                   style="width: 100%; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;"
                   value="{{if .Filter.AmountMin}}{{amountToDollars .Filter.AmountMin}}{{end}}">
        </div>

        <!-- Amount max filter -->
        <div>
            <label for="amount_max" style="display: block; margin-bottom: 0.25rem; font-weight: bold;">Amount Max</label>
            <input type="number"
                   name="amount_max"
                   id="amount_max"
                   step="0.01"
                   placeholder="0.00"
                   style="width: 100%; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;"
                   value="{{if .Filter.AmountMax}}{{amountToDollars .Filter.AmountMax}}{{end}}">
        </div>

        <!-- Date from filter -->
        <div>
            <label for="date_from" style="display: block; margin-bottom: 0.25rem; font-weight: bold;">Date From</label>
            <input type="date"
                   name="date_from"
                   id="date_from"
                   style="width: 100%; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;"
                   value="{{if .Filter.DateFrom}}{{formatDate .Filter.DateFrom}}{{end}}">
        </div>

        <!-- Date to filter -->
        <div>
            <label for="date_to" style="display: block; margin-bottom: 0.25rem; font-weight: bold;">Date To</label>
            <input type="date"
                   name="date_to"
                   id="date_to"
                   style="width: 100%; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;"
                   value="{{if .Filter.DateTo}}{{formatDate .Filter.DateTo}}{{end}}">
        </div>

        <!-- Sort options -->
        <div>
            <label for="sort" style="display: block; margin-bottom: 0.25rem; font-weight: bold;">Sort By</label>
            <select name="sort" id="sort" style="width: 100%; padding: 0.5rem; border: 1px solid #ccc; border-radius: 4px;">
                <option value="date:desc" {{if eq .Sort.String "date:desc"}}selected{{end}}>Date (Newest)</option>
                <option value="date:asc" {{if eq .Sort.String "date:asc"}}selected{{end}}>Date (Oldest)</option>
                <option value="amount:desc" {{if eq .Sort.String "amount:desc"}}selected{{end}}>Amount (High to Low)</option>
                <option value="amount:asc" {{if eq .Sort.String "amount:asc"}}selected{{end}}>Amount (Low to High)</option>
            </select>
        </div>
    </div>

    <!-- Actions -->
    <div style="margin-top: 1rem; display: flex; gap: 1rem;">
        <button type="submit" style="padding: 0.5rem 1rem; background-color: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;">Apply Filters</button>
        <a href="/expenses" style="padding: 0.5rem 1rem; background-color: #6c757d; color: white; border-radius: 4px; text-decoration: none; display: inline-block;">Clear All</a>
    </div>
</form>
```

**Step 3: Verify template syntax**

Run: `go build ./...`
Expected: Success (template will be validated at runtime)

---

## Task 14: Manual Testing

**Files:**
- None (testing only)

**Step 1: Build and run the application**

Run: `CGO_ENABLED=1 go build -o expensetrace . && ./expensetrace`

**Step 2: Test in browser**

1. Navigate to `http://localhost:8080/expenses`
2. Verify filter bar is visible
3. Test each filter individually:
   - Description search
   - Source search
   - Amount min/max
   - Date range
   - Sort options
4. Test combined filters
5. Verify URL updates with query params
6. Test "Clear All" button
7. Test bookmark/refresh preserves filters

**Step 3: Document any issues found**

Create notes of any bugs or visual issues to fix.

---

## Task 15: Fix Any Issues from Manual Testing

**Files:**
- TBD based on issues found

**Step 1: Review notes from manual testing**

**Step 2: Fix critical bugs**

(This will depend on what issues were found)

---

## Task 16: Update Handler Tests

**Files:**
- Modify: `internal/router/expense_test.go`

**Step 1: Add test for expensesHandler with filters**

Add to `internal/router/expense_test.go`:

```go
func TestExpensesHandlerWithFilters(t *testing.T) {
	// Setup test dependencies
	db := testutil.SetupTestDB(t)
	storage := sqlite.New(db)
	ctx := context.Background()

	// Create test user
	user, err := storage.CreateUser(ctx, "testuser", "hashedpass")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	userID := user.ID()

	// Insert test expenses
	expenses := []pkgStorage.Expense{
		pkgStorage.NewExpense(0, "visa", "morning coffee", "USD", -500, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), pkgStorage.ChargeType, nil),
		pkgStorage.NewExpense(0, "mastercard", "lunch", "USD", -1200, time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), pkgStorage.ChargeType, nil),
		pkgStorage.NewExpense(0, "visa", "afternoon coffee", "USD", -450, time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC), pkgStorage.ChargeType, nil),
	}
	_, err = storage.InsertExpenses(ctx, userID, expenses)
	if err != nil {
		t.Fatalf("failed to insert expenses: %v", err)
	}

	// Create session
	sessionID := "test-session-123"
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = storage.CreateSession(ctx, userID, sessionID, expiresAt)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Setup router
	logger := logger.New(logger.Config{Level: logger.LevelInfo, Format: logger.FormatText, Output: "stdout"})
	matcher := matcher.New([]pkgStorage.Category{})
	r := NewRouter(storage, matcher, logger)

	tests := []struct {
		name           string
		url            string
		wantStatus     int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:       "no filters returns all expenses",
			url:        "/expenses",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "morning coffee") {
					t.Error("expected to find 'morning coffee'")
				}
				if !strings.Contains(body, "lunch") {
					t.Error("expected to find 'lunch'")
				}
			},
		},
		{
			name:       "filter by description",
			url:        "/expenses?description=coffee",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "morning coffee") {
					t.Error("expected to find 'morning coffee'")
				}
				if !strings.Contains(body, "afternoon coffee") {
					t.Error("expected to find 'afternoon coffee'")
				}
				if strings.Contains(body, "lunch") {
					t.Error("did not expect to find 'lunch'")
				}
			},
		},
		{
			name:       "filter by source",
			url:        "/expenses?source=visa",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "morning coffee") {
					t.Error("expected to find 'morning coffee'")
				}
				if strings.Contains(body, "lunch") {
					t.Error("did not expect to find 'lunch'")
				}
			},
		},
		{
			name:       "invalid filter returns error",
			url:        "/expenses?amount_min=invalid",
			wantStatus: http.StatusOK, // Still renders page
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "Invalid filters") {
					t.Error("expected error message about invalid filters")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			req.AddCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			})

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.String())
			}
		})
	}
}
```

**Step 2: Run tests**

Run: `go test ./internal/router/... -v -run TestExpensesHandlerWithFilters`
Expected: PASS (may need adjustments based on actual template output)

---

## Task 17: Update Documentation

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Add section about filtering system**

Add to `CLAUDE.md` after the "Interface Design Philosophy" section:

```markdown
## Filtering and Sorting System

ExpenseTrace includes a flexible filtering and sorting system for expenses:

### Features
- **Persistent filter bar**: Always visible at the top of the expenses page
- **Filter fields**: Description, Source, Amount range (min/max), Date range (from/to)
- **Sorting**: By date or amount, ascending or descending
- **URL-based state**: Filters preserved in query parameters for bookmarking/sharing
- **AND logic**: All filters combine to narrow results

### Architecture

The filtering system uses the Specification Pattern with four layers:

1. **Filter Specification** (`internal/filter/filter.go`): Type-safe structs for filter criteria
2. **URL Parser** (`internal/filter/parser.go`): Converts query params to filter objects
3. **Storage Layer** (`internal/storage/sqlite/expenses.go`): Dynamic SQL generation
4. **Handler Layer** (`internal/router/expense.go`): Integration and state management

### Usage Examples

Filter by description:
```
/expenses?description=coffee
```

Filter by amount range:
```
/expenses?amount_min=5.00&amount_max=20.00
```

Filter by date range and sort:
```
/expenses?date_from=2024-01-01&date_to=2024-01-31&sort=amount:desc
```

Combined filters:
```
/expenses?description=coffee&source=visa&amount_max=10.00&sort=date:asc
```

### Implementation Details

- Filters use pointer fields to distinguish "not set" from zero values
- SQL queries built dynamically with prepared statements for security
- Default sort: date descending (newest first)
- All filters maintain user data isolation

### Testing

Comprehensive tests cover:
- URL parsing and validation (`internal/filter/`)
- SQL generation and filtering logic (`internal/storage/sqlite/`)
- Handler integration (`internal/router/`)
- Multi-user data isolation
```

---

## Task 18: Final Integration Test

**Files:**
- Create: `internal/integration/filter_test.go`

**Step 1: Create integration test directory**

Run: `mkdir -p internal/integration`

**Step 2: Write end-to-end integration test**

Create `internal/integration/filter_test.go`:

```go
package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/router"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/storage/sqlite"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

// TestFilteringEndToEnd tests the complete filtering flow from HTTP request to database query.
func TestFilteringEndToEnd(t *testing.T) {
	// Setup
	db := testutil.SetupTestDB(t)
	st := sqlite.New(db)
	ctx := context.Background()

	// Create user
	user, err := st.CreateUser(ctx, "testuser", "hashedpass")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	userID := user.ID()

	// Create session
	sessionID := "integration-test-session"
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = st.CreateSession(ctx, userID, sessionID, expiresAt)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

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

	_, err = st.InsertExpenses(ctx, userID, expenses)
	if err != nil {
		t.Fatalf("failed to insert expenses: %v", err)
	}

	// Setup router
	log := logger.New(logger.Config{Level: logger.LevelError, Format: logger.FormatText, Output: "stdout"})
	m := matcher.New([]storage.Category{})
	r := router.NewRouter(st, m, log)

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
			queryString:      "?amount_min=-1000&amount_max=-400",
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
			queryString:      "?description=coffee&source=visa&amount_max=-500",
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
			req.AddCookie(&http.Cookie{
				Name:  "session_id",
				Value: sessionID,
			})

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

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
```

**Step 3: Run integration test**

Run: `go test ./internal/integration/... -v`
Expected: PASS

---

## Task 19: Final Verification

**Files:**
- None (verification only)

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests PASS

**Step 2: Run linter**

Run: `make lint`
Expected: No errors (or fix any that appear)

**Step 3: Build application**

Run: `CGO_ENABLED=1 go build -o expensetrace .`
Expected: Successful build

**Step 4: Manual smoke test**

1. Run: `./expensetrace`
2. Navigate to http://localhost:8080
3. Test all filter combinations
4. Verify URL state persistence
5. Test bookmark functionality
6. Verify multi-user isolation (create second user)

**Step 5: Create summary of changes**

Document:
- Files created
- Files modified
- Tests added
- Features implemented

---

## Success Criteria Checklist

- [ ] Users can filter expenses by description, source, amount range, and date range
- [ ] Users can sort by date or amount in ascending/descending order
- [ ] Filter state persists in URL for bookmarking/sharing
- [ ] Filter bar always visible and easy to use
- [ ] All filters combine with AND logic
- [ ] "Clear All" resets to unfiltered view
- [ ] Code is well-tested (parser, storage, handlers)
- [ ] SQL injection protection verified
- [ ] Multi-user data isolation maintained
- [ ] All existing tests still pass
- [ ] Documentation updated
- [ ] No linter errors

---

## Notes for Implementation

**DRY Principles:**
- Reuse `extractExpensesFromRows` helper
- Reuse existing grouping logic in handler
- Share filter parsing logic across all filtered views (future)

**YAGNI:**
- No OR logic (not needed yet)
- No saved filter presets (not needed yet)
- No export filtered results (can add later)
- No uncategorized view filtering (postponed)

**TDD Workflow:**
- Write failing test first
- Run to verify failure
- Implement minimal code
- Run to verify pass
