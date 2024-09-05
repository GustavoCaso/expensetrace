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

type command struct {
	c       cli.Command
	flagSet *flag.FlagSet
}

var subcommands = map[string]*command{
	"delete": {
		c: delete.NewCommand(),
	},
	"category": {
		c: category.NewCommand(),
	},
	"import": {
		c: importCmd.NewCommand(),
	},
	"report": {
		c: report.NewCommand(),
	},
	"search": {
		c: search.NewCommand(),
	},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("subcommand is required\n")
		printUsage()

		os.Exit(1)
	}

	initFlagSets()

	commandName := os.Args[1]
	command, ok := subcommands[commandName]
	if ok {
		command.flagSet.Parse(os.Args[2:])

		conf, err := config.Parse(configPath)

		if err != nil {
			log.Fatalf("Unable to parse the configuration: %s", err.Error())
		}

		command.c.Run(conf)
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

	for commandName, cliCommand := range subcommands {
		fmt.Printf("subcommmand <%s>: %s\n", commandName, cliCommand.c.Description())
		cliCommand.flagSet.PrintDefaults()
		fmt.Println()
		fmt.Println()
	}
}

func printUsage() {
	fmt.Printf("usage: expensetrace <subcommand> [flags]\n\n")
}

func initFlagSets() {
	for commandName, cliCommand := range subcommands {
		fset := flag.NewFlagSet(commandName, flag.ExitOnError)
		fset.StringVar(&configPath, "c", "expense.yaml", "Configuration file")

		cliCommand.c.SetFlags(fset)
		cliCommand.flagSet = fset
	}
}
