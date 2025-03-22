package importCmd

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	importUtil "github.com/GustavoCaso/expensetrace/internal/import"
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
		return fmt.Errorf("you must provide a file to import")
	}

	file, err := os.Open(importFile)
	if err != nil {
		return fmt.Errorf("unable to open file: %w", err)
	}
	defer file.Close()

	errors := importUtil.Import(importFile, file, db, matcher)
	if len(errors) > 0 {
		var errMsg string
		for i, err := range errors {
			if i > 0 {
				errMsg += "; "
			}
			errMsg += err.Error()
		}
		return fmt.Errorf("unable to import file: %s", errMsg)
	}

	return nil
}
