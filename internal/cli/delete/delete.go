package delete

import (
	"flag"
	"log"
	"os"

	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/config"
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

func (c deleteCommand) Run(conf *config.Config) {
	err := expenseDB.DeleteExpenseDB(conf.DB)
	if err != nil {
		log.Fatalf("Unable to delete expense table: %s", err.Error())
	}

	os.Exit(0)
}
