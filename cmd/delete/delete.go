package main

import (
	"flag"
	"log"
	"os"

	"github.com/GustavoCaso/expensetrace/pkg/config"
	expenseDB "github.com/GustavoCaso/expensetrace/pkg/db"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "c", "expense.toml", "Configuration file")
	flag.Parse()

	conf, err := config.Parse(configPath)

	if err != nil {
		log.Fatalf("Unable to parse the configuration: %s", err.Error())
	}

	err = expenseDB.DeleteExpenseDB(conf.DB)
	if err != nil {
		log.Fatalf("Unable to delete expense table: %s", err.Error())
	}

	os.Exit(0)
}
