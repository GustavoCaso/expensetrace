package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/router"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

type webCommand struct {
}

func NewCommand() cli.Command {
	return webCommand{}
}

func (c webCommand) Description() string {
	return "Web interface"
}

const (
	defaultPort    = "8080"
	defaultTimeout = 5 * time.Second
)

func (c webCommand) Run(storage storage.Storage, matcher *category.Matcher, logger *logger.Logger) error {
	// Initialize configuration from environment variables
	port := os.Getenv("EXPENSETRACE_PORT")
	if port == "" {
		port = defaultPort
	}
	var timeout time.Duration

	customTimeout := os.Getenv("EXPENSETRACE_TIMEOUT")
	if customTimeout != "" {
		duration, err := time.ParseDuration(customTimeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse custom timeout, using default timeout of 5s")
			timeout = defaultTimeout
		} else {
			timeout = duration
		}
	} else {
		timeout = defaultTimeout
	}

	handler, _ := router.New(storage, matcher, logger)
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
