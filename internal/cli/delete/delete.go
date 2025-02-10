package delete

import (
	"database/sql"
	"flag"
	"log"
	"os"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
)

type deleteCommand struct {
}

func NewCommand() cli.Command {
	return deleteCommand{}
}

func (c deleteCommand) Description() string {
	return "Delete the expenses DB"
}

func (c deleteCommand) SetFlags(*flag.FlagSet) {
}

func (c deleteCommand) Run(db *sql.DB, _ *category.Matcher) {
	err := expenseDB.DeleteExpenseDB(db)
	if err != nil {
		log.Fatalf("Unable to delete expense table: %s", err.Error())
	}

	err = expenseDB.DeleteCategoriesDB(db)
	if err != nil {
		log.Fatalf("Unable to delete categories table: %s", err.Error())
	}

	os.Exit(0)
}
