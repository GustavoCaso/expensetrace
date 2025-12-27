package filter

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
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
