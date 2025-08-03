package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	categoryPkg "github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
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
			fmt.Fprintf(os.Stderr, "Unable to parse flag arguments. %s", err.Error())
		}

		conf, err := config.Parse(configPath)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to parse the configuration. %s", err.Error())
		}

		appLogger := logger.New(conf.Logger)

		appLogger.Info("Using database", "path", conf.DB)

		dbInstance, err := db.GetDB(conf.DB)
		if err != nil {
			appLogger.Fatal("Unable to get DB", "error", err.Error())
		}

		err = db.ApplyMigrations(dbInstance, appLogger)
		if err != nil {
			appLogger.Fatal("Unable to create schema", "error", err.Error())
		}

		err = db.PopulateCategoriesFromConfig(dbInstance, conf)

		if err != nil {
			appLogger.Error("Error inserting category", "error", err.Error())
		}

		categories, err := db.GetCategories(dbInstance)
		if err != nil {
			appLogger.Fatal("Unable to get categories", "error", err.Error())
		}

		matcher := categoryPkg.NewMatcher(categories)

		err = command.c.Run(dbInstance, matcher, appLogger)

		if err != nil {
			appLogger.Error("Command execution failed", "error", err)
			os.Exit(1)
		}

		err = dbInstance.Close()

		if err != nil {
			appLogger.Error("Error closing DB", "error", err)
			os.Exit(1)
		}

		os.Exit(0)
	}
	if strings.Contains(commandName, "help") {
		printHelp()

		os.Exit(0)
	}

	fmt.Fprintf(
		os.Stderr,
		"Unsupported command `%s`. Use 'help' command to print information about supported commands",
		commandName,
	)
	os.Exit(1)
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
