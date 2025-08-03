package main

import (
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

var subcommands = map[string]cli.Command{
	"tui": tui.NewCommand(),
	"web": web.NewCommand(),
}

const minArgsRequired = 2

func main() {
	if len(os.Args) < minArgsRequired {
		fmt.Printf("subcommand is required\n")
		printHelp()
		os.Exit(1)
	}

	commandName := os.Args[1]
	command, ok := subcommands[commandName]

	if ok {
		executeCommand(command)
		return
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

func executeCommand(command cli.Command) {
	configPath := os.Getenv("EXPENSETRACE_CONFIG")
	if configPath == "" {
		configPath = "expense.yml"
	}

	conf, err := config.Parse(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse the configuration. %s", err.Error())
		os.Exit(1)
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

	err = command.Run(dbInstance, matcher, appLogger)
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

func printHelp() {
	fmt.Printf("usage: expensetrace <subcommand>\n\n")
	fmt.Printf("Configuration is managed through environment variables and config file.\n")
	fmt.Printf("Use EXPENSETRACE_CONFIG to specify config file path (default: expense.yml)\n\n")

	for commandName, cliCommand := range subcommands {
		fmt.Printf("subcommmand <%s>: %s\n", commandName, cliCommand.Description())
	}
	fmt.Println()
}
