package main

import (
	"context"
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

	storage, err := sqlite.New(conf.GetDBConfig())
	if err != nil {
		appLogger.Fatal("Unable to get DB", "error", err.Error())
	}

	err = storage.ApplyMigrations(context.Background(), appLogger)
	if err != nil {
		appLogger.Fatal("Unable to create schema", "error", err.Error())
	}

	categories, err := storage.GetCategories(context.Background())
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
	fmt.Printf("Configuration is managed through environment variables and config file.\n\n")

	fmt.Printf("General:\n")
	fmt.Printf("  EXPENSETRACE_CONFIG              Config file path (default: expensetrace.yml)\n\n")

	fmt.Printf("Database:\n")
	fmt.Printf("  EXPENSETRACE_DB                  Database file path (default: expensetrace.db)\n")
	fmt.Printf("  EXPENSETRACE_DB_MAX_OPEN_CONNS   Maximum open connections (default: unlimited)\n")
	fmt.Printf("  EXPENSETRACE_DB_MAX_IDLE_CONNS   Maximum idle connections (default: 2)\n")
	fmt.Printf("  EXPENSETRACE_DB_CONN_MAX_LIFETIME Connection max lifetime (e.g., 1h, 30m)\n")
	fmt.Printf("  EXPENSETRACE_DB_CONN_MAX_IDLE_TIME Connection max idle time (e.g., 5m)\n")
	fmt.Printf("  EXPENSETRACE_DB_JOURNAL_MODE     SQLite journal mode (e.g., WAL, DELETE)\n")
	fmt.Printf("  EXPENSETRACE_DB_SYNCHRONOUS      SQLite synchronous mode (e.g., NORMAL, FULL)\n")
	fmt.Printf("  EXPENSETRACE_DB_CACHE_SIZE       SQLite cache size in KB (e.g., -2000)\n")
	fmt.Printf("  EXPENSETRACE_DB_BUSY_TIMEOUT     Busy timeout in milliseconds\n")
	fmt.Printf("  EXPENSETRACE_DB_WAL_AUTOCHECKPOINT WAL autocheckpoint interval\n")
	fmt.Printf("  EXPENSETRACE_DB_TEMP_STORE       Temp store location (MEMORY, FILE, DEFAULT)\n\n")

	fmt.Printf("Logging:\n")
	fmt.Printf("  EXPENSETRACE_LOG_LEVEL           Log level (default: info)\n")
	fmt.Printf("  EXPENSETRACE_LOG_FORMAT          Log format (default: text)\n")
	fmt.Printf("  EXPENSETRACE_LOG_OUTPUT          Log output (default: stdout)\n\n")

	fmt.Printf("Subcommands:\n")
	for commandName, cliCommand := range subcommands {
		fmt.Printf("  %-10s %s\n", commandName, cliCommand.Description())
	}
	fmt.Println()
}
