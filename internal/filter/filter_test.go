package filter

import (
	"net/url"
	"testing"
	"time"
)

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
		name        string
		queryString string
		wantFilter  *ExpenseFilter
		wantSort    *SortOptions
		wantErr     bool
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
