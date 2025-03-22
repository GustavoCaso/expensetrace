package delete

import (
	"database/sql"
	"flag"
	"fmt"

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

func (c deleteCommand) Run(db *sql.DB, _ *category.Matcher) error {
	err := expenseDB.DeleteExpenseDB(db)
	if err != nil {
		return fmt.Errorf("Unable to delete expense table: %w", err)
	}

	err = expenseDB.DeleteCategoriesDB(db)
	if err != nil {
		return fmt.Errorf("Unable to delete categories table: %s", err)
	}

	return nil
}
