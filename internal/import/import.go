package importutil

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/matcher"
	storageType "github.com/GustavoCaso/expensetrace/internal/storage"
)

type jsonExpense struct {
	Source      string    `json:"source"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
}

type ImportInfo struct {
	TotalImports          int
	ImportWithoutCategory []storageType.Expense
	ImportWithCategory    []storageType.Expense
	Error                 error
}

type entry struct {
	charge      bool
	source      string
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

func Import(
	ctx context.Context,
	filename string,
	reader io.Reader,
	storage storageType.Storage,
	categoryMatcher *matcher.Matcher,
) ImportInfo {
	info := ImportInfo{}
	expenses := []storageType.Expense{}

	fileFormat := path.Ext(filename)

	switch fileFormat {
	case ".csv":
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
			ex.source = source

			categoryID, category := categoryMatcher.Match(ex.description)
			var et storageType.ExpenseType
			if ex.amount < 0 {
				et = storageType.ChargeType
			} else {
				et = storageType.IncomeType
			}

			expense := storageType.NewExpense(
				0,
				ex.source,
				ex.description,
				ex.currency,
				ex.amount,
				ex.date,
				et,
				categoryID,
			)

			if category == "" {
				info.ImportWithoutCategory = append(info.ImportWithoutCategory, expense)
			} else {
				info.ImportWithCategory = append(info.ImportWithCategory, expense)
			}

			expenses = append(expenses, expense)
		}
	case ".json":
		e := []jsonExpense{}

		err := json.NewDecoder(reader).Decode(&e)
		if err != nil {
			info.Error = err
			return info
		}

		for _, jsonExp := range e {
			description := strings.ToLower(jsonExp.Description)
			categoryID, category := categoryMatcher.Match(description)

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

			if category == "" {
				info.ImportWithoutCategory = append(info.ImportWithoutCategory, expense)
			} else {
				info.ImportWithCategory = append(info.ImportWithCategory, expense)
			}

			expenses = append(expenses, expense)
		}

	default:
		info.Error = fmt.Errorf("unsupported file format: %s", fileFormat)
		return info
	}

	inserted, err := storage.InsertExpenses(ctx, expenses)

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
