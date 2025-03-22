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
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
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

func (c categoryCommand) Run(db *sql.DB, matcher *category.Matcher) error {
	var expenses []*expenseDB.Expense
	var err error
	if actionFlag == "migrate" {
		expenses, err = expenseDB.GetExpenses(db)
	} else {
		expenses, err = expenseDB.GetExpensesWithoutCategory(db)
	}
	if err != nil {
		return fmt.Errorf("unable to get expenses: %w", err)
	}

	switch actionFlag {
	case "inspect":
		var output io.Writer
		output = os.Stdout
		if outputLocation != "" {
			f, err := os.Create(outputLocation)
			if err != nil {
				return fmt.Errorf("unable to create inspect file output: %w", err)
			}

			output = f

			defer f.Close()
		}
		if err := inspect(output, expenses); err != nil {
			return err
		}
	case "recategorize", "migrate":
		if err := recategorize(db, matcher, expenses); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported action: %s", actionFlag)
	}

	return nil
}

func NewCommand() cli.Command {
	return categoryCommand{}
}

type reportExpense struct {
	count   int
	dates   []time.Time
	amounts []int64
}

func inspect(writer io.Writer, expenses []*expenseDB.Expense) error {
	if len(expenses) == 0 {
		log.Println("No expenses without category 🎉")
		return nil
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

	return nil
}

func recategorize(db *sql.DB, categoryMatcher *category.Matcher, expenses []*expenseDB.Expense) error {
	expensesToUpdate := []*expenseDB.Expense{}
	for _, ex := range expenses {
		id, c := categoryMatcher.Match(ex.Description)

		if id > 0 && c != "" {
			ex.CategoryID = id
			expensesToUpdate = append(expensesToUpdate, ex)
		}
	}

	if len(expensesToUpdate) > 0 {
		updated, err := expenseDB.UpdateExpenses(db, expensesToUpdate)

		if err != nil {
			return fmt.Errorf("unexpected error updating categories: %w", err)
		}

		if updated != int64(len(expensesToUpdate)) {
			log.Printf("Not all records were updated :(")
		}

		log.Printf("%d updated\n", updated)
	} else {
		log.Println("No expenses that could recategorize")
	}

	return nil
}
