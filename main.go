package main

import (
	"flag"
	"log"
	"os"

	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/cli/category"
	"github.com/GustavoCaso/expensetrace/internal/cli/delete"
	importCmd "github.com/GustavoCaso/expensetrace/internal/cli/import"
	"github.com/GustavoCaso/expensetrace/internal/cli/report"
	"github.com/GustavoCaso/expensetrace/internal/config"
)

var subcommands = map[string]cli.Command{
	"delete":   delete.NewCommand(),
	"category": category.NewCommand(),
	"import":   importCmd.NewCommand(),
	"report":   report.NewCommand(),
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("subcommand is required")
	}

	commandName := os.Args[1]
	command, ok := subcommands[commandName]
	if ok {
		var configPath string

		fset := flag.NewFlagSet(commandName, flag.ExitOnError)
		fset.StringVar(&configPath, "c", "expense.toml", "Configuration file")

		command.SetFlags(fset)
		fset.Parse(os.Args[2:])

		conf, err := config.Parse(configPath)

		if err != nil {
			log.Fatalf("Unable to parse the configuration: %s", err.Error())
		}

		command.Run(conf)
	} else {
		log.Fatalf("unsupported comand %s", commandName)
	}
}
