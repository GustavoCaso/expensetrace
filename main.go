package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/cli/category"
	"github.com/GustavoCaso/expensetrace/internal/cli/delete"
	importCmd "github.com/GustavoCaso/expensetrace/internal/cli/import"
	"github.com/GustavoCaso/expensetrace/internal/cli/report"
	"github.com/GustavoCaso/expensetrace/internal/cli/search"
	"github.com/GustavoCaso/expensetrace/internal/config"
)

var configPath string

var subcommands = map[string]cli.Command{
	"delete":   delete.NewCommand(),
	"category": category.NewCommand(),
	"import":   importCmd.NewCommand(),
	"report":   report.NewCommand(),
	"search":   search.NewCommand(),
}

var subcommandsFlagSets = map[string]*flag.FlagSet{
	"delete":   nil,
	"category": nil,
	"import":   nil,
	"report":   nil,
	"search":   nil,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("subcommand is required\n")
		printUsage()

		os.Exit(1)
	}

	for c, cLogic := range subcommands {
		fset := flag.NewFlagSet(c, flag.ExitOnError)
		fset.StringVar(&configPath, "c", "expense.toml", "Configuration file")

		cLogic.SetFlags(fset)

		subcommandsFlagSets[c] = fset
	}

	commandName := os.Args[1]
	command, ok := subcommands[commandName]
	if ok {

		subcommandsFlagSets[commandName].Parse(os.Args[2:])

		conf, err := config.Parse(configPath)

		if err != nil {
			log.Fatalf("Unable to parse the configuration: %s", err.Error())
		}

		command.Run(conf)
	} else {
		if strings.Contains(commandName, "help") {
			printHelp()

			os.Exit(0)
		}
		log.Fatalf("unsupported comand %s. \nUse 'help' command to print information about supported commands\n", commandName)
	}
}

func printHelp() {
	printUsage()

	for c, cLogic := range subcommands {
		fmt.Printf("subcommmand <%s>: %s\n", c, cLogic.Description())
		subcommandsFlagSets[c].PrintDefaults()
		fmt.Println()
		fmt.Println()
	}
}

func printUsage() {
	fmt.Printf("usage: expensetrace <subcommand> [flags]\n\n")
}
