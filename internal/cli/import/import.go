package importcmd

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	importUtil "github.com/GustavoCaso/expensetrace/internal/import"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

type importCommand struct {
}

func NewCommand() cli.Command {
	return importCommand{}
}

func (c importCommand) Description() string {
	return "Imports expenses to the DB"
}

var importFile string

func (c importCommand) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&importFile, "f", "", "file to import")
}

func (c importCommand) Run(db *sql.DB, matcher *category.Matcher) error {
	if importFile == "" {
		return errors.New("you must provide a file to import")
	}

	file, err := os.Open(importFile)
	if err != nil {
		return err
	}
	defer file.Close()

	info := importUtil.Import(importFile, file, db, matcher)
	if info.Error != nil && info.TotalImports == 0 {
		return fmt.Errorf("unable to import expenses due to error: %w", info.Error)
	}

	if info.TotalImports > 0 {
		fmt.Printf("Total expenses imported: %d\n", info.TotalImports)
	} else {
		fmt.Println("No expenses were imported")
	}
	if len(info.ImportWithoutCategory) > 0 {
		fmt.Print("The following expenses were imported without a category: \n")
		for _, expense := range info.ImportWithoutCategory {
			fmt.Printf("  - %s: %s\n", expense.Description, util.FormatMoney(expense.Amount, ".", ","))
		}

	}
	if info.Error != nil {
		fmt.Printf("Errors importing file: %s\n", info.Error)
	}

	return nil
}
