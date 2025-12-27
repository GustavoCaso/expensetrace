# Expense Filtering and Sorting Feature Design

**Date:** 2025-12-27
**Status:** Approved

## Overview

This design document describes the implementation of a flexible filtering and sorting system for the ExpenseTrace application's expenses view. The system uses a Specification Pattern approach with type-safe filter objects and reusable architecture.

## Requirements

### Expenses View
- **Persistent filter bar** at the top of the page (always visible)
- **Filter fields**: Description, Source, Amount range (min/max), Date range (from/to)
- **Sorting options**: Date (asc/desc), Amount (asc/desc)
- **Filter logic**: AND-only (all filters must match)
- **URL persistence**: Filters reflected in query parameters for bookmarking/sharing

### Categories View
- No filtering or sorting needed (keep simple)

### Architecture Goals
- Reusable filter framework that can be extended to other entities
- Type-safe filter specifications with IDE support
- Clean separation of concerns (parsing → specification → SQL)
- Testable components

## Architecture

### Component Overview

The filtering system consists of four layers:

```
URL (?description=coffee&sort=date:desc)
  → Handler parses query params
  → ParseExpenseFilters() creates filter objects
  → Storage.GetExpensesFiltered() builds & executes SQL
  → Handler renders results + preserves filter state in form
```

#### 1. Filter Specification Layer (`internal/filter/`)
- `ExpenseFilter` struct: Type-safe holder for filter criteria
- `SortOptions` struct: Type-safe holder for sort preferences
- Validation methods

#### 2. URL Parsing Layer (`internal/filter/parser.go`)
- `ParseExpenseFilters(url.Values)`: Converts query params to filter objects
- Type conversion and validation
- Error handling for bad input

#### 3. Storage/Repository Layer (`internal/storage/sqlite/expenses.go`)
- `GetExpensesFiltered()`: Dynamic SQL generation based on filter fields
- Prepared statements for security
- Reuses existing helper functions

#### 4. Handler Layer (`internal/router/expense.go`)
- Modified `expensesHandler`: Integrates filter parsing and state management
- Passes filter state back to templates

## Detailed Design

### 1. Filter Specification Structs

**Location:** `internal/filter/filter.go`

```go
type ExpenseFilter struct {
    Description *string    // LIKE search on description
    Source      *string    // LIKE search on source
    AmountMin   *int64     // Minimum amount in cents (inclusive)
    AmountMax   *int64     // Maximum amount in cents (inclusive)
    DateFrom    *time.Time // Start date (inclusive)
    DateTo      *time.Time // End date (inclusive)
}

type SortOptions struct {
    Field     SortField     // What to sort by
    Direction SortDirection // ASC or DESC
}

type SortField string
const (
    SortByDate   SortField = "date"
    SortByAmount SortField = "amount"
)

type SortDirection string
const (
    SortAsc  SortDirection = "asc"
    SortDesc SortDirection = "desc"
)

func DefaultSortOptions() *SortOptions {
    return &SortOptions{
        Field:     SortByDate,
        Direction: SortDesc, // Newest first
    }
}
```

**Design decisions:**
- Pointer fields distinguish "not set" from empty/zero values
- Enums (string consts) prevent invalid sort values
- Default sort ensures consistent behavior

### 2. URL Parsing Layer

**Location:** `internal/filter/parser.go`

```go
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

    // Parse amount range (convert to cents)
    if min := params.Get("amount_min"); min != "" {
        val, err := parseAmount(min)
        if err != nil {
            return nil, nil, fmt.Errorf("invalid amount_min: %w", err)
        }
        filter.AmountMin = &val
    }
    // Similar for amount_max...

    // Parse date range
    if from := params.Get("date_from"); from != "" {
        val, err := time.Parse("2006-01-02", from)
        if err != nil {
            return nil, nil, fmt.Errorf("invalid date_from: %w", err)
        }
        filter.DateFrom = &val
    }
    // Similar for date_to...

    // Parse sort (format: "date:desc" or "amount:asc")
    if s := params.Get("sort"); s != "" {
        parsed, err := parseSort(s)
        if err != nil {
            return nil, nil, fmt.Errorf("invalid sort: %w", err)
        }
        sort = parsed
    }

    return filter, sort, nil
}
```

**Helper functions:**
- `parseAmount(string) (int64, error)`: Converts "10.50" → 1050 cents
- `parseSort(string) (*SortOptions, error)`: Parses "date:desc" format

**URL Examples:**
- `/expenses?description=coffee&sort=amount:desc`
- `/expenses?date_from=2024-01-01&date_to=2024-01-31&amount_min=5.00`
- `/expenses?source=visa&sort=date:asc`

### 3. Repository/Storage Layer

**New interface method in `internal/storage/storage.go`:**

```go
GetExpensesFiltered(ctx context.Context, userID int64, filter *filter.ExpenseFilter, sort *filter.SortOptions) ([]Expense, error)
```

**Implementation in `internal/storage/sqlite/expenses.go`:**

```go
func (s *sqliteStorage) GetExpensesFiltered(
    ctx context.Context,
    userID int64,
    filter *filter.ExpenseFilter,
    sort *filter.SortOptions,
) ([]Expense, error) {
    query := "SELECT * FROM expenses WHERE user_id = ?"
    args := []interface{}{userID}

    // Add filters dynamically
    if filter.Description != nil {
        query += " AND description LIKE ?"
        args = append(args, "%"+*filter.Description+"%")
    }

    if filter.Source != nil {
        query += " AND source LIKE ?"
        args = append(args, "%"+*filter.Source+"%")
    }

    if filter.AmountMin != nil {
        query += " AND amount >= ?"
        args = append(args, *filter.AmountMin)
    }

    if filter.AmountMax != nil {
        query += " AND amount <= ?"
        args = append(args, *filter.AmountMax)
    }

    if filter.DateFrom != nil {
        query += " AND date >= ?"
        args = append(args, filter.DateFrom.Unix())
    }

    if filter.DateTo != nil {
        query += " AND date <= ?"
        args = append(args, filter.DateTo.Unix())
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

**Key features:**
- Always filters by `user_id` for data isolation
- Dynamic WHERE clauses only for set filters
- Parameterized queries prevent SQL injection
- Reuses existing `extractExpensesFromRows()` helper
- Default sort always applied

### 4. Handler Layer

**Modified `expensesHandler` in `internal/router/expense.go`:**

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

**Updated view data:**

```go
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

**Route changes:**
- Remove `POST /expense/search` (replaced by filter bar)
- Remove old `expenseSearchHandler` function
- `GET /expenses` handles both filtered and unfiltered views

### 5. Template Layer

**Filter bar in `web/templates/pages/expenses/index.html`:**

```html
<!-- Filter Bar - Always visible at top -->
<form method="GET" action="/expenses" class="filter-bar">
    <div class="filter-row">
        <!-- Description filter -->
        <div class="filter-field">
            <label for="description">Description</label>
            <input type="text"
                   name="description"
                   id="description"
                   placeholder="Search description..."
                   value="{{if .Filter.Description}}{{.Filter.Description}}{{end}}">
        </div>

        <!-- Source filter -->
        <div class="filter-field">
            <label for="source">Source</label>
            <input type="text"
                   name="source"
                   id="source"
                   placeholder="Search source..."
                   value="{{if .Filter.Source}}{{.Filter.Source}}{{end}}">
        </div>

        <!-- Amount range -->
        <div class="filter-field">
            <label for="amount_min">Amount Min</label>
            <input type="number"
                   name="amount_min"
                   id="amount_min"
                   step="0.01"
                   placeholder="0.00"
                   value="{{if .Filter.AmountMin}}{{amountToDollars .Filter.AmountMin}}{{end}}">
        </div>

        <div class="filter-field">
            <label for="amount_max">Amount Max</label>
            <input type="number"
                   name="amount_max"
                   id="amount_max"
                   step="0.01"
                   placeholder="0.00"
                   value="{{if .Filter.AmountMax}}{{amountToDollars .Filter.AmountMax}}{{end}}">
        </div>

        <!-- Date range -->
        <div class="filter-field">
            <label for="date_from">Date From</label>
            <input type="date"
                   name="date_from"
                   id="date_from"
                   value="{{if .Filter.DateFrom}}{{formatDate .Filter.DateFrom}}{{end}}">
        </div>

        <div class="filter-field">
            <label for="date_to">Date To</label>
            <input type="date"
                   name="date_to"
                   id="date_to"
                   value="{{if .Filter.DateTo}}{{formatDate .Filter.DateTo}}{{end}}">
        </div>

        <!-- Sort options -->
        <div class="filter-field">
            <label for="sort">Sort By</label>
            <select name="sort" id="sort">
                <option value="date:desc" {{if eq .Sort.String "date:desc"}}selected{{end}}>Date (Newest)</option>
                <option value="date:asc" {{if eq .Sort.String "date:asc"}}selected{{end}}>Date (Oldest)</option>
                <option value="amount:desc" {{if eq .Sort.String "amount:desc"}}selected{{end}}>Amount (High to Low)</option>
                <option value="amount:asc" {{if eq .Sort.String "amount:asc"}}selected{{end}}>Amount (Low to High)</option>
            </select>
        </div>

        <!-- Actions -->
        <div class="filter-actions">
            <button type="submit">Apply Filters</button>
            <a href="/expenses" class="clear-filters">Clear All</a>
        </div>
    </div>
</form>
```

**Required template helpers:**
- `amountToDollars`: Convert cents to dollar string (1050 → "10.50")
- `formatDate`: Format time.Time to "2006-01-02"
- `.Sort.String()`: Method on SortOptions returning "field:direction"

**Behavior:**
- Form submits via GET, updating URL query params
- "Clear All" navigates to `/expenses` with no params
- Filter values pre-populated from current state

## Testing Strategy

### 1. Filter Parser Tests (`internal/filter/parser_test.go`)

Test URL parsing and validation scenarios:
- All filters present
- No filters (defaults)
- Individual filters
- Invalid formats (amount, date, sort)
- Edge cases (negative amounts, future dates, etc.)

**Coverage goal:** 100% (critical for security)

### 2. Storage Layer Tests (`internal/storage/sqlite/expenses_test.go`)

Test SQL generation and filtering logic:
- Individual filter fields
- Combined filters (AND logic)
- Amount range filtering
- Date range filtering
- Sort by date ascending/descending
- Sort by amount ascending/descending
- Empty filter (returns all user's expenses)
- Multi-user data isolation

### 3. Handler Integration Tests (`internal/router/expense_test.go`)

Test end-to-end HTTP requests:
- Valid filters return correct results
- Invalid filters display error message
- URL state preservation
- Filter bar pre-population
- Clear filters functionality

### Test Data Strategy

- Create diverse expense test data (various dates, amounts, descriptions, sources)
- Test multi-user scenarios to ensure data isolation
- Include edge cases (very large amounts, very old dates, special characters)

## Implementation Plan

### Phase 1: Foundation
1. Create `internal/filter/` package
2. Define `ExpenseFilter` and `SortOptions` structs
3. Implement `DefaultSortOptions()`
4. Write unit tests for filter structs

### Phase 2: Parsing Layer
5. Implement `ParseExpenseFilters()` function
6. Implement helper functions (`parseAmount`, `parseSort`, etc.)
7. Write comprehensive parser tests
8. Test error handling

### Phase 3: Storage Layer
9. Add `GetExpensesFiltered()` to Storage interface
10. Implement SQL generation logic in SQLite storage
11. Write storage layer tests
12. Verify SQL injection protection

### Phase 4: Handler Layer
13. Modify `expensesHandler` to use filters
14. Update `expesesViewData` struct
15. Remove old search handler
16. Write handler integration tests

### Phase 5: Template Layer
17. Create filter bar HTML
18. Implement template helpers
19. Add CSS styling for filter bar
20. Test form submission and state preservation

### Phase 6: Polish
21. Add "Clear All" functionality
22. Ensure accessibility (labels, tab order)
23. Test on different screen sizes
24. Documentation updates

## Future Enhancements

This architecture enables future filtering on:
- Categories (if needed later)
- Reports
- Uncategorized expenses view
- Any new entities

The `internal/filter/` package can be extended with new filter types (e.g., `CategoryFilter`, `ReportFilter`) following the same pattern.

## Migration Notes

**Breaking changes:**
- `POST /expense/search` route removed (replaced by filter bar)
- `expenseSearchHandler` function removed
- `Query` field in `expesesViewData` replaced by `Filter` and `Sort`

**Backward compatibility:**
- Existing `/expenses` GET route continues to work (shows all expenses)
- No database schema changes required
- No impact on other features

## Success Criteria

- [ ] Users can filter expenses by description, source, amount range, and date range
- [ ] Users can sort by date or amount in ascending/descending order
- [ ] Filter state persists in URL for bookmarking/sharing
- [ ] Filter bar always visible and easy to use
- [ ] All filters combine with AND logic
- [ ] "Clear All" resets to unfiltered view
- [ ] Code is well-tested (parser, storage, handlers)
- [ ] SQL injection protection verified
- [ ] Multi-user data isolation maintained
