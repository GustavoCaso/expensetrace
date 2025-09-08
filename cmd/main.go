package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/cli/tui"
	"github.com/GustavoCaso/expensetrace/internal/cli/web"
	"github.com/GustavoCaso/expensetrace/internal/config"
	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage/sqlite"
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
		configPath = "expensetrace.yml"
	}

	conf, err := config.Parse(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse the configuration. %s", err.Error())
		os.Exit(1)
	}

	appLogger := logger.New(conf.Logger)

	appLogger.Info("Using database", "path", conf.DB)

	storage, err := sqlite.New(conf.DB)
	if err != nil {
		appLogger.Fatal("Unable to get DB", "error", err.Error())
	}

	err = storage.ApplyMigrations(appLogger)
	if err != nil {
		appLogger.Fatal("Unable to create schema", "error", err.Error())
	}

	categories, err := storage.GetCategories()
	if err != nil {
		appLogger.Fatal("Unable to get categories", "error", err.Error())
	}

	matcher := matcher.New(categories)

	err = command.Run(storage, matcher, appLogger)
	if err != nil {
		appLogger.Error("Command execution failed", "error", err)
		os.Exit(1)
	}

	err = storage.Close()
	if err != nil {
		appLogger.Error("Error closing storage", "error", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func printHelp() {
	fmt.Printf("usage: expensetrace <subcommand>\n\n")
	fmt.Printf("Configuration is managed through environment variables and config file.\n")
	fmt.Printf("Use EXPENSETRACE_CONFIG to specify config file path (default: expensetrace.yml)\n")
	fmt.Printf("Use EXPENSETRACE_DB to specify db file (default: expensetrace.db)\n")
	fmt.Printf("Use EXPENSETRACE_LOG_LEVEL to specify log level (default: info)\n")
	fmt.Printf("Use EXPENSETRACE_LOG_FORMAT to specify log format (default: text)\n")
	fmt.Printf("Use EXPENSETRACE_LOG_OUTPUT to specify log output (default: stdout)\n\n")

	for commandName, cliCommand := range subcommands {
		fmt.Printf("subcommmand <%s>: %s\n", commandName, cliCommand.Description())
	}
	fmt.Println()
}
