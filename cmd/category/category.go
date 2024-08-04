package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"golang.org/x/exp/maps"

	"github.com/GustavoCaso/expensetrace/pkg/category"
	expenseDB "github.com/GustavoCaso/expensetrace/pkg/db"
	"github.com/GustavoCaso/expensetrace/pkg/expense"
)

func main() {
	var actionFlag string
	var outputLocation string
	flag.StringVar(&actionFlag, "a", "inspect", "What action to perform. Supported values are: inspect, reprocess")
	flag.StringVar(&outputLocation, "o", "", "Where to print the inspect output result")
	flag.Parse()

	db, err := expenseDB.GetOrCreateExpenseDB()
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
		reprocess(db, expenses)
	default:
		log.Fatalf("Unsupported action: %s", actionFlag)
	}
}

func inspect(writer io.Writer, expenses []expense.Expense) {
	if len(expenses) == 0 {
		log.Println("No expenses without category 🎉")
		os.Exit(0)
	}

	groupedExpenses := map[string]int{}

	for _, ex := range expenses {
		groupedExpenses[ex.Description]++
	}

	keys := maps.Keys(groupedExpenses)

	sort.SliceStable(keys, func(i, j int) bool {
		return groupedExpenses[keys[i]] > groupedExpenses[keys[j]]
	})

	for _, k := range keys {
		fmt.Fprintf(writer, "%s -> %d\n", k, groupedExpenses[k])
	}

	os.Exit(0)
}

func reprocess(db *sql.DB, expenses []expense.Expense) {
	expensesToUpdate := []expense.Expense{}
	for _, ex := range expenses {
		c := category.Match(ex.Description)

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
