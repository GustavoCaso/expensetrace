package category

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

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/config"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/expense"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

var actionFlag string
var outputLocation string

type categoryCommand struct {
}

func (c categoryCommand) Description() string {
	return "Allows to interact with the expenses category."
}

func (c categoryCommand) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&actionFlag, "a", "inspect", "What action to perform. Supported values are: inspect, recategorize, migrate")
	fs.StringVar(&outputLocation, "o", "", "Where to print the inspect output result")
}

func (c categoryCommand) Run(conf *config.Config, db *sql.DB) {
	defer db.Close()

	var expenses []expense.Expense
	var err error
	if actionFlag == "migrate" {
		expenses, err = expenseDB.GetExpenses(db)
	} else {
		expenses, err = expenseDB.GetExpensesWithoutCategory(db)
	}
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
	case "recategorize", "migrate":
		categoryMatcher := category.New(conf.Categories)

		recategorize(db, categoryMatcher, expenses)
	default:
		log.Fatalf("Unsupported action: %s", actionFlag)
	}
}

func NewCommand() cli.Command {
	return categoryCommand{}
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
			r.amounts = append(r.amounts, ex.AmountWithSign())
			groupedExpenses[ex.Description] = r
		} else {
			groupedExpenses[ex.Description] = reportExpense{
				count: 1,
				dates: []time.Time{
					ex.Date,
				},
				amounts: []int64{
					ex.AmountWithSign(),
				},
			}
		}
	}

	keys := maps.Keys(groupedExpenses)

	sort.SliceStable(keys, func(i, j int) bool {
		return groupedExpenses[keys[i]].count > groupedExpenses[keys[j]].count
	})

	var total int

	for _, k := range keys {
		expense := groupedExpenses[k]
		count := expense.count
		fmt.Fprintf(writer, "%s -> %d\n", k, count)
		total += count

		for i, date := range groupedExpenses[k].dates {
			fmt.Fprintf(writer, "	[%s] %s€\n", date.Format("2006-01-02"), util.FormatMoney(expense.amounts[i], ".", ","))
		}
	}

	fmt.Fprint(writer, "\n")

	fmt.Fprintf(writer, "There are a total of %d uncategorized expenses", total)

	os.Exit(0)
}

func recategorize(db *sql.DB, categoryMatcher category.Category, expenses []expense.Expense) {
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
