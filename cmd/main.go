package main

import (
	"context"
	"fmt"
	"os"

	"github.com/GustavoCaso/expensetrace/internal/cli/web"
	"github.com/GustavoCaso/expensetrace/internal/config"
	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/storage/sqlite"
)

func main() {
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

	err = storage.ApplyMigrations(context.Background(), appLogger)
	if err != nil {
		appLogger.Fatal("Unable to create schema", "error", err.Error())
	}

	err = web.Run(storage, appLogger)
	if err != nil {
		appLogger.Error("failed to run the expensetrace web service", "error", err)
		os.Exit(1)
	}

	err = storage.Close()
	if err != nil {
		appLogger.Error("Error closing storage", "error", err)
		os.Exit(1)
	}

	os.Exit(0)
}
