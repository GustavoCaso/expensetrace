package expense

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/GustavoCaso/expensetrace/domain"
	storageType "github.com/GustavoCaso/expensetrace/storage"
)

const (
	centsToDecimal = 100.0
	decimalPlaces  = 2
	base10         = 10
)

// csvExport exports expenses to CSV format
// format: ID,Source,Date,Description,Amount,Type,Currency,Category
func csvExport(
	ctx context.Context,
	userID int64,
	writer io.Writer,
	expenses []domain.Expense,
	storage storageType.Storage,
) error {
	w := csv.NewWriter(writer)
	defer w.Flush()

	// Pre-allocate records slice: header + all expense records
	records := make([][]string, 0, len(expenses)+1)

	// Add header
	header := []string{"ID", "Source", "Date", "Description", "Amount", "Type", "Currency", "Category"}
	records = append(records, header)

	// Convert all expenses to CSV records
	for _, expense := range expenses {
		records = append(records, expenseToCSVRecord(ctx, userID, expense, storage))
	}

	// Write all records at once
	if err := w.WriteAll(records); err != nil {
		return fmt.Errorf("failed to write CSV records: %w", err)
	}

	return nil
}

func expenseToCSVRecord(
	ctx context.Context,
	userID int64,
	expense domain.Expense,
	storage storageType.Storage,
) []string {
	// Get category name if category exists
	categoryName := ""
	if expense.CategoryID() != nil {
		category, err := storage.GetCategory(ctx, userID, *expense.CategoryID())
		if err == nil {
			categoryName = category.Name()
		}
	}

	// Format amount (convert from cents to decimal)
	amountFloat := float64(expense.Amount()) / centsToDecimal
	amountStr := strconv.FormatFloat(amountFloat, 'f', decimalPlaces, 64)

	// Format date
	dateStr := expense.Date().Format("2006-01-02")

	// Format type
	typeStr := "charge"
	if expense.Type() == domain.IncomeType {
		typeStr = "income"
	}

	record := []string{
		strconv.FormatInt(expense.ID(), base10),
		expense.Source(),
		dateStr,
		expense.Description(),
		amountStr,
		typeStr,
		expense.Currency(),
		categoryName,
	}

	return record
}
