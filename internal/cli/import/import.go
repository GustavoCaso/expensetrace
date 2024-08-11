package importCmd

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/config"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/expense"
)

var re = regexp.MustCompile(`(?P<charge>-)?(?P<amount>\d+)\.?(?P<decimal>\d*)`)
var chargeIndex = re.SubexpIndex("charge")
var amountIndex = re.SubexpIndex("amount")
var decimalIndex = re.SubexpIndex("decimal")

type importCommand struct {
}

func NewCommand() cli.Command {
	return importCommand{}
}

func (c importCommand) SetFlags(*flag.FlagSet) {
}

func (c importCommand) Run(conf *config.Config) {
	argsLength := len(os.Args)

	if argsLength != 2 {
		panic("must provide a CSV file with your expenseses")
	}

	categoryMatcher := category.New(conf.Categories)

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

			parsedAmount, err := strconv.ParseInt(fmt.Sprintf("%s%s", amount, decimal), 10, 64)
			if err != nil {
				log.Fatal(err)
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
		log.Fatalf("Unsupported file format: %s", fileFormat)
		os.Exit(1)
	}

	db, err := expenseDB.GetOrCreateExpenseDB(conf.DB)
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
