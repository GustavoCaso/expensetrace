package main

import (
	"database/sql"
	"flag"
	"log"
	"os"

	expenseDB "github.com/GustavoCaso/sandbox/go/moneyTracker/pkg/db"
	"github.com/GustavoCaso/sandbox/go/moneyTracker/pkg/expense"
)

func main() {
	var actionFlag string
	flag.StringVar(&actionFlag, "a", "inspect", "What action to perform. Supported values are: inspect, reprocess")

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
		reprocess(db)
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

func reprocess(db *sql.DB) {
	os.Exit(0)
}
