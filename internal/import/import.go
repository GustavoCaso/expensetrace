package importutil

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/matcher"
	storageType "github.com/GustavoCaso/expensetrace/internal/storage"
)

var re = regexp.MustCompile(`(?P<charge>-)?(?P<amount>\d+)\.?(?P<decimal>\d*)`)
var chargeIndex = re.SubexpIndex("charge")
var amountIndex = re.SubexpIndex("amount")
var decimalIndex = re.SubexpIndex("decimal")

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

func Import(
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
		// CSV format
		// source,date,description,amount,currency
		// source: string
		// date: dd/mm/yyyy
		// description: string
		// amount: string it can include minus signs
		// currency: string
		r := csv.NewReader(reader)
		for {
			record, err := r.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				info.Error = err
				return info
			}

			t, err := time.Parse("02/01/2006", record[1])
			if err != nil {
				info.Error = err
				return info
			}

			description := strings.ToLower(record[2])
			categoryID, category := categoryMatcher.Match(description)

			matches := re.FindStringSubmatch(record[3])
			if len(matches) == 0 {
				info.Error = fmt.Errorf("amount regex did not find any matches for %s", record[3])
				return info
			}

			amount := matches[amountIndex]
			decimal := matches[decimalIndex]
			sign := matches[chargeIndex]

			// Parse the full amount string including decimal and sign
			// If there's no sign, it's a positive number
			amountStr := fmt.Sprintf("%s%s", amount, decimal)
			if sign == "-" {
				amountStr = "-" + amountStr
			}
			parsedAmount, err := strconv.ParseInt(amountStr, 10, 64)
			if err != nil {
				info.Error = err
				return info
			}

			var et storageType.ExpenseType
			if parsedAmount < 0 {
				et = storageType.ChargeType
			} else {
				et = storageType.IncomeType
			}

			expense := storageType.NewExpense(
				0,
				record[0],
				description,
				record[4],
				parsedAmount,
				t,
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

	inserted, err := storage.InsertExpenses(expenses)

	info.TotalImports = int(inserted)
	if err != nil {
		info.Error = fmt.Errorf("unexpected error inserting expenses: %w", err)
	}

	return info
}
