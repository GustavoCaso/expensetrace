package importCmd

import (
	"flag"
	"log"
	"os"
	"path"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/config"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
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

func (c importCommand) Run(conf *config.Config) {
	fileFormat := path.Ext(importFile)
	file, err := os.Open(importFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	categoryMatcher := category.New(conf.Categories)

	db, err := expenseDB.GetOrCreateExpenseDB(conf.DB)
	if err != nil {
		log.Fatalf("Unable to get expenses DB: %s", err.Error())
		os.Exit(1)
	}

	defer db.Close()

	errors := importUtil.Import(fileFormat, file, db, categoryMatcher)

	if len(errors) > 0 {
		log.Println("Unable to import expenses, errors:")
		for _, err := range errors {
			log.Println(err.Error())
		}

		os.Exit(1)
	}

	os.Exit(0)
}
