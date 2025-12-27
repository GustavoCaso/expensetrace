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
