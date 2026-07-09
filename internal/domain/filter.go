package domain

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

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

func parseAmount(s string) (int64, error) {
	if s == "" {
		return 0, errors.New("amount cannot be empty")
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format: %w", err)
	}

	cents := int64(f * 100) //nolint:mnd // the value is obvious
	return cents, nil
}

// parseSort parses a sort string like "date:desc" into SortOptions.
func parseSort(s string) (*SortOptions, error) {
	if s == "" {
		return nil, errors.New("sort string cannot be empty")
	}

	parts := strings.Split(s, ":")
	if len(parts) != 2 { //nolint:mnd // the value is obvious
		return nil, errors.New("invalid sort format, expected field:direction")
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
