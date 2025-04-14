package deletecmd

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
	err := expenseDB.DropTables(db)
	if err != nil {
		return fmt.Errorf("unable to delete tables: %w", err)
	}

	return nil
}
