package delete

import (
	"flag"
	"log"
	"os"

	"github.com/GustavoCaso/expensetrace/pkg/command"
	"github.com/GustavoCaso/expensetrace/pkg/config"
	expenseDB "github.com/GustavoCaso/expensetrace/pkg/db"
)

type deleteCommand struct {
}

func NewCommand() command.Command {
	return deleteCommand{}
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
