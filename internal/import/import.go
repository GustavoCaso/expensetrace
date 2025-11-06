package importutil

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/matcher"
	storageType "github.com/GustavoCaso/expensetrace/internal/storage"
)

type JSONExpense struct {
	Source      string    `json:"source"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
}

type ImportInfo struct {
	TotalImports          int
	ImportWithoutCategory int
	Error                 error
}

type entry struct {
	charge      bool
	date        time.Time
	description string
	amount      int64
	currency    string
}

type transformer func(v string, entry *entry) error

var defaultSourceTransformers = map[string][]transformer{
	"evo":       evoTransformers,
	"revolut":   revolutTransformers,
	"bankinter": bankinterTransformers,
}

var availableSources = slices.Sorted(maps.Keys(defaultSourceTransformers))

func SupportedProvider(filename string) bool {
	source, err := extractFileSource(filename)
	if err != nil {
		return false
	}
	_, ok := defaultSourceTransformers[source]
	return ok
}

func SupportedJSONSchema(reader io.Reader) (bool, []JSONExpense) {
	var expenses []JSONExpense

	// Try to unmarshal into expected type
	if err := json.NewDecoder(reader).Decode(&expenses); err != nil {
		return false, expenses
	}

	// Validate that it's not empty and has required fields
	if len(expenses) == 0 {
		return false, expenses
	}

	// Validate first expense has all required fields
	first := expenses[0]
	if first.Source == "" || first.Description == "" ||
		first.Currency == "" || first.Date.IsZero() {
		return false, expenses
	}

	return true, expenses
}

func ImportJSON(ctx context.Context, userID int64, expenses []JSONExpense, storage storageType.Storage,
	categoryMatcher *matcher.Matcher) ImportInfo {
	info := ImportInfo{}
	storageExpenses := []storageType.Expense{}

	for _, jsonExp := range expenses {
		description := strings.ToLower(jsonExp.Description)
		categoryID, _ := categoryMatcher.Match(description)

		var et storageType.ExpenseType
		if jsonExp.Amount < 0 {
			et = storageType.ChargeType
		} else {
			et = storageType.IncomeType
		}

		expense := storageType.NewExpense(
			0,
			jsonExp.Source,
			description,
			jsonExp.Currency,
			jsonExp.Amount,
			jsonExp.Date,
			et,
			categoryID,
		)

		if categoryID == nil {
			info.ImportWithoutCategory++
		}

		storageExpenses = append(storageExpenses, expense)
	}

	inserted, err := storage.InsertExpenses(ctx, userID, storageExpenses)

	info.TotalImports = int(inserted)
	if err != nil {
		info.Error = fmt.Errorf("unexpected error inserting expenses: %w", err)
	}

	return info
}

func ImportCSV(
	ctx context.Context,
	userID int64,
	filename string,
	reader io.Reader,
	storage storageType.Storage,
	categoryMatcher *matcher.Matcher,
) ImportInfo {
	info := ImportInfo{}
	expenses := []storageType.Expense{}

	source, err := extractFileSource(filename)

	if err != nil {
		info.Error = err
		return info
	}

	transformerFuncs, ok := defaultSourceTransformers[source]
	if !ok {
		info.Error = fmt.Errorf(
			"no source transformer avilable for %s. Available sources: %s",
			source,
			availableSources,
		)
		return info
	}

	captilizedSource := withTitleCase(source)

	r := csv.NewReader(reader)

	// Read all records
	records, err := r.ReadAll()
	if err != nil {
		info.Error = err
		return info
	}

	startRow := 1 // Skip header row

	// Process each record
	for i := startRow; i < len(records); i++ {
		record := records[i]

		ex := &entry{}
		for idxcol, f := range transformerFuncs {
			if f == nil {
				// skip this col
				continue
			}

			value := record[idxcol]
			if value != "" {
				tranformerErr := f(value, ex)
				if tranformerErr != nil {
					info.Error = tranformerErr
					return info
				}
			}
		}

		categoryID, _ := categoryMatcher.Match(ex.description)
		var et storageType.ExpenseType
		if ex.amount < 0 {
			et = storageType.ChargeType
		} else {
			et = storageType.IncomeType
		}

		expense := storageType.NewExpense(
			0,
			captilizedSource,
			ex.description,
			ex.currency,
			ex.amount,
			ex.date,
			et,
			categoryID,
		)

		if categoryID == nil {
			info.ImportWithoutCategory++
		}

		expenses = append(expenses, expense)
	}

	inserted, err := storage.InsertExpenses(ctx, userID, expenses)

	info.TotalImports = int(inserted)
	if err != nil {
		info.Error = fmt.Errorf("unexpected error inserting expenses: %w", err)
	}

	return info
}

func extractFileSource(filename string) (string, error) {
	parts := strings.Split(filename, "_")
	if len(parts) <= 1 {
		return "", fmt.Errorf(
			"no able to extract source from filename. Use filename with format <source>_*.csv. Available sources: %s",
			availableSources,
		)
	}
	return parts[0], nil
}

func withTitleCase(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}
