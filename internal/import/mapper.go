package importutil

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/matcher"
	storageType "github.com/GustavoCaso/expensetrace/internal/storage"
)

var amountRe = regexp.MustCompile(`(?P<charge>-)?(?P<amount>\d+)\.?(?P<decimal>\d*)`)
var chargeIdx = amountRe.SubexpIndex("charge")
var amountIdx = amountRe.SubexpIndex("amount")
var decimalIdx = amountRe.SubexpIndex("decimal")

// FieldMapping defines how file columns map to expense fields.
type FieldMapping struct {
	Source            string // Manual source input (e.g., "Chase Bank")
	DateColumn        int    // Index of date column
	DescriptionColumn int    // Index of description column
	AmountColumn      int    // Index of amount column
	CurrencyColumn    int    // Index of currency column
}

// mappingError represents an error that occurred while mapping a specific row.
type mappingError struct {
	RowIndex int
	Error    error
}

// MappingResult contains the results of applying a field mapping.
type MappingResult struct {
	Expenses []storageType.Expense
	Errors   []mappingError
}

// Validate checks if the field mapping is valid.
func (m *FieldMapping) Validate(headerCount int) error {
	if m.Source == "" {
		return errors.New("source is required")
	}
	if m.DateColumn < 0 || m.DateColumn >= headerCount {
		return fmt.Errorf("invalid date column index: %d", m.DateColumn)
	}
	if m.DescriptionColumn < 0 || m.DescriptionColumn >= headerCount {
		return fmt.Errorf("invalid description column index: %d", m.DescriptionColumn)
	}
	if m.AmountColumn < 0 || m.AmountColumn >= headerCount {
		return fmt.Errorf("invalid amount column index: %d", m.AmountColumn)
	}
	if m.CurrencyColumn < 0 || m.CurrencyColumn >= headerCount {
		return fmt.Errorf("invalid currency column index: %d", m.CurrencyColumn)
	}
	return nil
}

// ApplyMapping applies the field mapping to parsed data and creates expenses.
func ApplyMapping(
	data *ParsedData,
	mapping *FieldMapping,
	categoryMatcher *matcher.Matcher,
) (*MappingResult, error) {
	if err := mapping.Validate(len(data.Headers)); err != nil {
		return nil, fmt.Errorf("invalid mapping: %w", err)
	}

	result := &MappingResult{
		Expenses: make([]storageType.Expense, 0, len(data.Rows)),
		Errors:   make([]mappingError, 0),
	}

	for i, row := range data.Rows {
		expense, err := mapRow(row, mapping, categoryMatcher)
		if err != nil {
			result.Errors = append(result.Errors, mappingError{
				RowIndex: i,
				Error:    err,
			})
			continue
		}
		result.Expenses = append(result.Expenses, expense)
	}

	return result, nil
}

// mapRow converts a single row to an expense using the field mapping.
func mapRow(
	row []string,
	mapping *FieldMapping,
	categoryMatcher *matcher.Matcher,
) (storageType.Expense, error) {
	// Extract values from row
	source := mapping.Source // Use manual source input
	dateStr := row[mapping.DateColumn]
	description := strings.ToLower(row[mapping.DescriptionColumn])
	amountStr := row[mapping.AmountColumn]
	currency := row[mapping.CurrencyColumn]

	// Parse date
	date, err := parseDate(dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: %w", dateStr, err)
	}

	// Parse amount
	amount, err := parseAmount(amountStr)
	if err != nil {
		return nil, fmt.Errorf("invalid amount %q: %w", amountStr, err)
	}

	// Determine expense type
	var expenseType storageType.ExpenseType
	if amount < 0 {
		expenseType = storageType.ChargeType
	} else {
		expenseType = storageType.IncomeType
	}

	// Match category
	categoryID, _ := categoryMatcher.Match(description)

	// Create expense
	expense := storageType.NewExpense(
		0,
		source,
		description,
		currency,
		amount,
		date,
		expenseType,
		categoryID,
	)

	return expense, nil
}

const defaultDateFormat = "02/01/2006"

var fallbackFormats = []string{
	"2006-01-02",           // ISO format
	"01/02/2006",           // MM/DD/YYYY
	"2006-01-02T15:04:05Z", // ISO with time
}

// parseDate attempts to parse a date string using the specified format.
func parseDate(dateStr string) (time.Time, error) {
	t, err := time.Parse(defaultDateFormat, dateStr)
	if err == nil {
		return t, nil
	}

	for _, fallback := range fallbackFormats {
		if parsed, parseErr := time.Parse(fallback, dateStr); parseErr == nil {
			return parsed, nil
		}
	}

	return time.Time{}, errors.New("unable to parse date")
}

// parseAmount parses an amount string that may include signs, decimals, and formatting.
func parseAmount(amountStr string) (int64, error) {
	// Remove common formatting characters
	cleaned := strings.ReplaceAll(amountStr, ",", "")
	cleaned = strings.TrimSpace(cleaned)

	// Use the existing regex pattern from import.go
	matches := amountRe.FindStringSubmatch(cleaned)

	if len(matches) == 0 {
		return 0, errors.New("amount does not match expected pattern")
	}

	amount := matches[amountIdx]
	decimal := matches[decimalIdx]
	sign := matches[chargeIdx]

	// Combine amount and decimal
	amountStr = fmt.Sprintf("%s%s", amount, decimal)
	if sign == "-" {
		amountStr = "-" + amountStr
	}

	parsedAmount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse amount: %w", err)
	}

	return parsedAmount, nil
}
