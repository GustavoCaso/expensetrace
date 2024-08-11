package main

import (
	"flag"
	"log"
	"os"

	"github.com/GustavoCaso/expensetrace/cmd/category"
	"github.com/GustavoCaso/expensetrace/cmd/delete"
	importCmd "github.com/GustavoCaso/expensetrace/cmd/import"
	"github.com/GustavoCaso/expensetrace/cmd/report"
	"github.com/GustavoCaso/expensetrace/pkg/command"
	"github.com/GustavoCaso/expensetrace/pkg/config"
)

var subcommands = map[string]command.Command{
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
