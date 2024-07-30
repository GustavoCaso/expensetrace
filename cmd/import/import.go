package main

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoCaso/sandbox/go/moneyTracker/pkg/category"
	expenseDB "github.com/GustavoCaso/sandbox/go/moneyTracker/pkg/db"
	"github.com/GustavoCaso/sandbox/go/moneyTracker/pkg/expense"
)

var re = regexp.MustCompile(`(?P<charge>-)?(?P<amount>\d+)\.?(?P<decimal>\d*)`)
var chargeIndex = re.SubexpIndex("charge")
var amountIndex = re.SubexpIndex("amount")
var decimalIndex = re.SubexpIndex("decimal")

func main() {
	argsLength := len(os.Args)

	if argsLength != 2 {
		panic("must provide a CSV file with your expenseses")
	}

	expenseFile := os.Args[1]

	fileFormat := path.Ext(expenseFile)
	expenses := []expense.Expense{}
	switch fileFormat {
	case ".csv":
		file, err := os.Open(expenseFile)
		if err != nil {
			log.Fatal(err)
		}
		r := csv.NewReader(file)
		for {
			record, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}

			if strings.Contains(record[0], "Fecha") {
				// We skip the first line of the CSV
				continue
			}

			t, err := time.Parse("02/01/2006", record[1])
			if err != nil {
				log.Fatal(err)
			}

			matches := re.FindStringSubmatch(record[3])
			if len(matches) == 0 {
				log.Fatal("Amount regex did not find any matches")
			}

			var et expense.ExpenseType
			if matches[chargeIndex] == "-" {
				et = expense.ChargeType
			} else {
				et = expense.IncomeType
			}

			amount := matches[amountIndex]
			decimal := matches[decimalIndex]

			parsedAmount, err := strconv.ParseUint(amount, 10, 32)
			if err != nil {
				log.Fatal(err)
			}
			parsedDecimal, err := strconv.ParseUint(decimal, 10, 16)
			if err != nil {
				log.Fatal(err)
			}

			description := strings.ToLower(record[2])
			c := category.Match(description)

			if c == "" {
				log.Printf("expense without category. Description: %s\n", description)
			}

			expense := expense.Expense{
				Date:        t,
				Description: description,
				Amount:      uint32(parsedAmount),
				Decimal:     uint16(parsedDecimal),
				Type:        et,
				Currency:    record[4],
				Category:    c,
			}

			expenses = append(expenses, expense)
		}

	default:
		log.Fatalf("Unsupported file format: %s", fileFormat)
		os.Exit(1)
	}

	db, err := expenseDB.GetOrCreateExpenseDB()
	if err != nil {
		log.Fatalf("Unable to get expenses DB: %s", err.Error())
		os.Exit(1)
	}

	defer db.Close()

	err = expenseDB.InsertExpenses(db, expenses)
	if err != nil {
		log.Fatalf("Unable to import expenses: %s", err.Error())
		os.Exit(1)
	}

	os.Exit(0)
}
