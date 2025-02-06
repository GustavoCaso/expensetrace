package importUtil

import (
	"database/sql"
	"encoding/csv"
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
	"github.com/GustavoCaso/expensetrace/internal/expense"
)

var re = regexp.MustCompile(`(?P<charge>-)?(?P<amount>\d+)\.?(?P<decimal>\d*)`)
var chargeIndex = re.SubexpIndex("charge")
var amountIndex = re.SubexpIndex("amount")
var decimalIndex = re.SubexpIndex("decimal")

func Import(filename string, reader io.Reader, db *sql.DB, categoryMatcher category.Category) []error {
	errors := []error{}
	expenses := []expense.Expense{}

	fileFormat := path.Ext(filename)

	switch fileFormat {
	case ".csv":
		r := csv.NewReader(reader)
		for {
			record, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				errors = append(errors, err)
				return errors
			}

			if strings.Contains(record[0], "Fecha") {
				// We skip the first line of the CSV
				continue
			}

			t, err := time.Parse("02/01/2006", record[1])
			if err != nil {
				errors = append(errors, err)
				return errors
			}

			matches := re.FindStringSubmatch(record[3])
			if len(matches) == 0 {
				errors = append(errors, fmt.Errorf("amount regex did not find any matches"))
				return errors

			}

			var et expense.ExpenseType
			if matches[chargeIndex] == "-" {
				et = expense.ChargeType
			} else {
				et = expense.IncomeType
			}

			amount := matches[amountIndex]
			decimal := matches[decimalIndex]

			parsedAmount, err := strconv.ParseInt(fmt.Sprintf("%s%s", amount, decimal), 10, 64)
			if err != nil {
				errors = append(errors, err)
				return errors
			}

			description := strings.ToLower(record[2])
			c := categoryMatcher.Match(description)

			if c == "" {
				log.Printf("expense without category. Description: %s\n", description)
			}

			expense := expense.Expense{
				Date:        t,
				Description: description,
				Amount:      parsedAmount,
				Type:        et,
				Currency:    record[4],
				Category:    c,
			}

			expenses = append(expenses, expense)
		}

	default:
		errors = append(errors, fmt.Errorf("unsupported file format: %s", fileFormat))
		return errors
	}

	return expenseDB.InsertExpenses(db, expenses)
}
