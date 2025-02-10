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
)

var re = regexp.MustCompile(`(?P<charge>-)?(?P<amount>\d+)\.?(?P<decimal>\d*)`)
var chargeIndex = re.SubexpIndex("charge")
var amountIndex = re.SubexpIndex("amount")
var decimalIndex = re.SubexpIndex("decimal")

func Import(filename string, reader io.Reader, db *sql.DB, categoryMatcher *category.Matcher) []error {
	errors := []error{}
	expenses := []expenseDB.Expense{}

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

			var et expenseDB.ExpenseType
			if matches[chargeIndex] == "-" {
				et = expenseDB.ChargeType
			} else {
				et = expenseDB.IncomeType
			}

			amount := matches[amountIndex]
			decimal := matches[decimalIndex]

			parsedAmount, err := strconv.ParseInt(fmt.Sprintf("%s%s", amount, decimal), 10, 64)
			if err != nil {
				errors = append(errors, err)
				return errors
			}

			description := strings.ToLower(record[2])
			id, category := categoryMatcher.Match(description)

			if category == "" {
				log.Printf("expense without category. Description: %s\n", description)
			}

			expense := expenseDB.Expense{
				Date:        t,
				Description: description,
				Amount:      parsedAmount,
				Type:        et,
				Currency:    record[4],
				CategoryID:  id,
			}

			expenses = append(expenses, expense)
		}

	default:
		errors = append(errors, fmt.Errorf("unsupported file format: %s", fileFormat))
		return errors
	}

	return expenseDB.InsertExpenses(db, expenses)
}
