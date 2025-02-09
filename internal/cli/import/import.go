package importCmd

import (
	"database/sql"
	"flag"
	"log"
	"os"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/config"
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

func (c importCommand) Run(conf *config.Config, db *sql.DB) {
	file, err := os.Open(importFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	categoryMatcher := category.New(conf.Categories)

	defer db.Close()

	errors := importUtil.Import(importFile, file, db, categoryMatcher)

	if len(errors) > 0 {
		log.Println("Unable to import expenses, errors:")
		for _, err := range errors {
			log.Println(err.Error())
		}

		os.Exit(1)
	}

	os.Exit(0)
}
