package main

import (
	"flag"
	"fmt"
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
	"github.com/GustavoCaso/expensetrace/internal/logger"
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

	minFlagsNum := 2

	if len(os.Args) < minFlagsNum {
		fmt.Printf("subcommand is required\n")
		printHelp()

		os.Exit(1)
	}

	commandName := os.Args[1]
	command, ok := subcommands[commandName]
	//nolint:nestif // No need to extract this code to a function as is clear
	if ok {
		err := command.flagSet.Parse(os.Args[2:])
		if err != nil {
			logger.Fatal("Unable to parse flag arguments", "error", err.Error())
		}

		conf, err := config.Parse(configPath)

		if err != nil {
			logger.Fatal("Unable to parse the configuration", "error", err.Error())
		}

		appLogger := logger.New(conf.Logger)
		logger.SetDefault(appLogger)

		logger.Info("Using database", "path", conf.DB)

		dbInstance, err := db.GetDB(conf.DB)
		if err != nil {
			logger.Fatal("Unable to get DB", "error", err.Error())
		}

		err = db.ApplyMigrations(dbInstance)
		if err != nil {
			logger.Fatal("Unable to create schema", "error", err.Error())
		}

		err = db.PopulateCategoriesFromConfig(dbInstance, conf)

		if err != nil {
			logger.Error("Error inserting category", "error", err.Error())
		}

		categories, err := db.GetCategories(dbInstance)
		if err != nil {
			logger.Fatal("Unable to get categories", "error", err.Error())
		}

		matcher := categoryPkg.NewMatcher(categories)

		err = command.c.Run(dbInstance, matcher)

		if err != nil {
			logger.Error("Command execution failed", "error", err)
			os.Exit(1)
		}

		err = dbInstance.Close()

		if err != nil {
			logger.Error("Error closing DB", "error", err)
			os.Exit(1)
		}

		os.Exit(0)
	}
	if strings.Contains(commandName, "help") {
		printHelp()

		os.Exit(0)
	}
	logger.Fatal(
		"Unsupported command. Use 'help' command to print information about supported commands",
		"command", commandName,
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
