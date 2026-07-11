package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/config"
	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/router"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/storage/sqlite"
)

func main() {
	conf := config.Parse()

	appLogger := logger.New(conf.Logger)

	appLogger.Info("Using database", "path", conf.DBFile)

	storage, err := sqlite.New(conf.DBFile)
	if err != nil {
		appLogger.Fatal("Unable to get DB", "error", err.Error())
	}

	err = storage.ApplyMigrations(context.Background(), appLogger)
	if err != nil {
		appLogger.Fatal("Unable to create schema", "error", err.Error())
	}

	err = run(conf.Port, conf.Timeout, storage, appLogger)
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

func run(port string, timeout time.Duration, storage storage.Storage, logger *logger.Logger) error {
	handler := router.New(storage, logger)
	logger.Info("Starting web server", "url", fmt.Sprintf("http://localhost:%s", port))

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		ReadHeaderTimeout: timeout,
		Handler:           handler,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Unexpected server error", "error", err)
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan

	logger.Info("Received shutdown signal, shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := server.Shutdown(ctx)
	if err != nil {
		logger.Error("Issue shutting down server", "error", err)
	}
	return nil
}
