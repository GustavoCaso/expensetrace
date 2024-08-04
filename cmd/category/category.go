package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"time"

	"golang.org/x/exp/maps"

	"github.com/GustavoCaso/expensetrace/pkg/category"
	"github.com/GustavoCaso/expensetrace/pkg/config"
	expenseDB "github.com/GustavoCaso/expensetrace/pkg/db"
	"github.com/GustavoCaso/expensetrace/pkg/expense"
	"github.com/GustavoCaso/expensetrace/pkg/util"
)

func main() {
	var actionFlag string
	var outputLocation string
	var configPath string
	flag.StringVar(&actionFlag, "a", "inspect", "What action to perform. Supported values are: inspect, reprocess")
	flag.StringVar(&outputLocation, "o", "", "Where to print the inspect output result")
	flag.StringVar(&configPath, "c", "expense.toml", "Configuration file")
	flag.Parse()

	conf, err := config.Parse(configPath)

	if err != nil {
		log.Fatalf("Unable to parse the configuration: %s", err.Error())
	}

	db, err := expenseDB.GetOrCreateExpenseDB(conf.DB)
	if err != nil {
		log.Fatalf("Unable to get expenses DB: %s", err.Error())
		os.Exit(1)
	}

	defer db.Close()

	expenses, err := expenseDB.GetExpensesWithoutCategory(db)
	if err != nil {
		log.Fatalf("Unable to get expenses: %s", err.Error())
		os.Exit(1)
	}

	switch actionFlag {
	case "inspect":
		var output io.Writer
		output = os.Stdout
		if outputLocation != "" {
			f, err := os.Create(outputLocation)
			if err != nil {
				log.Fatalf("Unable to create inspect file output: %s", err.Error())
			}

			output = f

			defer f.Close()
		}
		inspect(output, expenses)
	case "reprocess":
		categoryMatcher := category.New(conf.Categories)

		reprocess(db, categoryMatcher, expenses)
	default:
		log.Fatalf("Unsupported action: %s", actionFlag)
	}
}

type reportExpense struct {
	count   int
	dates   []time.Time
	amounts []int64
}

func inspect(writer io.Writer, expenses []expense.Expense) {
	if len(expenses) == 0 {
		log.Println("No expenses without category 🎉")
		os.Exit(0)
	}

	groupedExpenses := map[string]reportExpense{}

	for _, ex := range expenses {
		if r, ok := groupedExpenses[ex.Description]; ok {
			r.count++
			r.dates = append(r.dates, ex.Date)
			r.amounts = append(r.amounts, ex.Amount)
			groupedExpenses[ex.Description] = r
		} else {
			groupedExpenses[ex.Description] = reportExpense{
				count: 1,
				dates: []time.Time{
					ex.Date,
				},
				amounts: []int64{
					ex.Amount,
				},
			}
		}
	}

	keys := maps.Keys(groupedExpenses)

	sort.SliceStable(keys, func(i, j int) bool {
		return groupedExpenses[keys[i]].count > groupedExpenses[keys[j]].count
	})

	for _, k := range keys {
		fmt.Fprintf(writer, "%s -> %d\n", k, groupedExpenses[k].count)
		for i, date := range groupedExpenses[k].dates {
			fmt.Fprintf(writer, "	%s %s€\n", date.Format("2006-01-02"), util.FormatMoney(groupedExpenses[k].amounts[i], ".", ","))
		}
	}

	os.Exit(0)
}

func reprocess(db *sql.DB, categoryMatcher category.Category, expenses []expense.Expense) {
	expensesToUpdate := []expense.Expense{}
	for _, ex := range expenses {
		c := categoryMatcher.Match(ex.Description)

		if c != "" {
			ex.Category = c
			expensesToUpdate = append(expensesToUpdate, ex)
		}
	}

	if len(expensesToUpdate) > 0 {
		updated, err := expenseDB.UpdateExpenses(db, expensesToUpdate)

		if err != nil {
			log.Fatalf("Unexpected error updating categories: %v", err)
		}

		if updated != int64(len(expensesToUpdate)) {
			log.Printf("Not all records were updated :(")
		}

		log.Printf("%d updated\n", updated)
	} else {
		log.Println("No expenses that could recategorize")
	}

	os.Exit(0)
}
