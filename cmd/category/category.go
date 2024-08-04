package main

import (
	"database/sql"
	"flag"
	"log"
	"os"

	"github.com/GustavoCaso/expensetrace/pkg/category"
	expenseDB "github.com/GustavoCaso/expensetrace/pkg/db"
	"github.com/GustavoCaso/expensetrace/pkg/expense"
)

func main() {
	var actionFlag string
	flag.StringVar(&actionFlag, "a", "inspect", "What action to perform. Supported values are: inspect, reprocess")
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
		inspect(expenses)
	case "reprocess":
		reprocess(db, expenses)
	default:
		log.Fatalf("Unsupported action: %s", actionFlag)
	}
}

func inspect(expenses []expense.Expense) {
	if len(expenses) == 0 {
		log.Println("No expenses without category 🎉")
		os.Exit(0)
	}

	log.Println("This are the expenses descriptions that do not have a category")
	for _, ex := range expenses {
		log.Printf("- %s\n", ex.Description)
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
