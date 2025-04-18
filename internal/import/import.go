package importutil

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
)

var re = regexp.MustCompile(`(?P<charge>-)?(?P<amount>\d+)\.?(?P<decimal>\d*)`)
var chargeIndex = re.SubexpIndex("charge")
var amountIndex = re.SubexpIndex("amount")
var decimalIndex = re.SubexpIndex("decimal")

type expense struct {
	Source      string    `json:"source"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
}

func Import(filename string, reader io.Reader, db *sql.DB, categoryMatcher *category.Matcher) []error {
	errs := []error{}
	expenses := []*expenseDB.Expense{}

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
				errs = append(errs, err)
				return errs
			}

			t, err := time.Parse("02/01/2006", record[1])
			if err != nil {
				errs = append(errs, err)
				return errs
			}

			description := strings.ToLower(record[2])
			id, category := categoryMatcher.Match(description)

			if category == "" {
				log.Printf("expense without category. Description: %s\n", description)
			}

			matches := re.FindStringSubmatch(record[3])
			if len(matches) == 0 {
				errs = append(errs, errors.New("amount regex did not find any matches"))
				return errs
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
				errs = append(errs, err)
				return errs
			}

			var et expenseDB.ExpenseType
			if parsedAmount < 0 {
				et = expenseDB.ChargeType
			} else {
				et = expenseDB.IncomeType
			}

			expense := &expenseDB.Expense{
				Source:      record[0],
				Date:        t,
				Description: description,
				Amount:      parsedAmount,
				Type:        et,
				Currency:    record[4],
				CategoryID:  id,
			}

			expenses = append(expenses, expense)
		}
	case ".json":
		e := []expense{}

		err := json.NewDecoder(reader).Decode(&e)
		if err != nil {
			errs = append(errs, err)
			return errs
		}

		for _, expense := range e {
			description := strings.ToLower(expense.Description)
			id, category := categoryMatcher.Match(description)

			if category == "" {
				log.Printf("expense without category. Description: %s\n", description)
			}

			var et expenseDB.ExpenseType
			if expense.Amount < 0 {
				et = expenseDB.ChargeType
			} else {
				et = expenseDB.IncomeType
			}

			expense := &expenseDB.Expense{
				Source:      expense.Source,
				Date:        expense.Date,
				Description: description,
				Amount:      expense.Amount,
				Type:        et,
				Currency:    expense.Currency,
				CategoryID:  id,
			}

			expenses = append(expenses, expense)
		}

	default:
		errs = append(errs, fmt.Errorf("unsupported file format: %s", fileFormat))
		return errs
	}

	return expenseDB.InsertExpenses(db, expenses)
}
