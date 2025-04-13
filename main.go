package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	categoryPkg "github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/cli/category"
	deleteCmd "github.com/GustavoCaso/expensetrace/internal/cli/delete"
	importCmd "github.com/GustavoCaso/expensetrace/internal/cli/import"
	"github.com/GustavoCaso/expensetrace/internal/cli/report"
	"github.com/GustavoCaso/expensetrace/internal/cli/search"
	"github.com/GustavoCaso/expensetrace/internal/cli/tui"
	"github.com/GustavoCaso/expensetrace/internal/cli/web"
	"github.com/GustavoCaso/expensetrace/internal/config"
	"github.com/GustavoCaso/expensetrace/internal/db"
)

var configPath string

type command struct {
	c       cli.Command
	flagSet *flag.FlagSet
}

var subcommands = map[string]*command{
	"delete": {
		c: deleteCmd.NewCommand(),
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
	"tui": {
		c: tui.NewCommand(),
	},
	"web": {
		c: web.NewCommand(),
	},
}

func main() {
	initFlagSets()

	if len(os.Args) < 2 {
		fmt.Printf("subcommand is required\n")
		printHelp()

		os.Exit(1)
	}

	commandName := os.Args[1]
	command, ok := subcommands[commandName]
	if ok {
		err := command.flagSet.Parse(os.Args[2:])
		if err != nil {
			log.Fatalf("Unable to parse flag arguments: %s", err.Error())
		}

		conf, err := config.Parse(configPath)

		if err != nil {
			log.Fatalf("Unable to parse the configuration: %s", err.Error())
		}

		log.Printf("Using db located at %s\n", conf.DB)

		dbInstance, err := db.GetDB(conf.DB)
		if err != nil {
			log.Fatalf("Unable to get DB: %s", err.Error())
		}

		err = db.ApplyMigrations(dbInstance)
		if err != nil {
			log.Fatalf("Unable to get create schema: %s", err.Error())
		}

		err = db.PopulateCategoriesFromConfig(dbInstance, conf)

		if err != nil {
			log.Printf("error inserting category. err: %v\n", err.Error())
		}

		categories, err := db.GetCategories(dbInstance)
		if err != nil {
			log.Fatalf("Unable to get categories: %s", err.Error())
		}

		matcher := categoryPkg.NewMatcher(categories)

		err = command.c.Run(dbInstance, matcher)
		dbInstance.Close()

		if err != nil {
			log.Printf("Error: %v", err)
			os.Exit(1)
		}

		os.Exit(0)
	}
	if strings.Contains(commandName, "help") {
		printHelp()

		os.Exit(0)
	}
	log.Fatalf(
		"unsupported comand %s. \nUse 'help' command to print information about supported commands\n",
		commandName,
	)
}

func printHelp() {
	fmt.Printf("usage: expensetrace <subcommand> [flags]\n\n")

	for commandName, cliCommand := range subcommands {
		fmt.Printf("subcommmand <%s>: %s\n", commandName, cliCommand.c.Description())
		cliCommand.flagSet.PrintDefaults()
		fmt.Println()
		fmt.Println()
	}
}

func initFlagSets() {
	for commandName, cliCommand := range subcommands {
		fset := flag.NewFlagSet(commandName, flag.ExitOnError)
		configPath = os.Getenv("EXPENSETRACE_CONFIG")
		if configPath == "" {
			configPath = "expense.yml"
		}
		fset.StringVar(&configPath, "c", configPath, "Configuration file")

		cliCommand.c.SetFlags(fset)
		cliCommand.flagSet = fset
	}
}
