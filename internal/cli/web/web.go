package web

import (
	"context"
	"database/sql"
	"errors"
	"flag"
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
)

type webCommand struct {
}

func NewCommand() cli.Command {
	return webCommand{}
}

func (c webCommand) Description() string {
	return "Web interface"
}

var port string
var timeout time.Duration
var allowEmbedding = false

const (
	defaultPort    = "8080"
	defaultTimeout = 5 * time.Second
)

func (c webCommand) SetFlags(fs *flag.FlagSet) {
	port = os.Getenv("EXPENSETRACE_PORT")
	if port == "" {
		port = defaultPort
	}
	customTimeout := os.Getenv("EXPENSETRACE_TIMEOUT")
	if customTimeout != "" {
		duration, err := time.ParseDuration(customTimeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse custom timeout, using default timeout of 5s")
		}
		timeout = duration
	} else {
		timeout = defaultTimeout
	}

	allowEmbedding = os.Getenv("EXPENSETRACE_ALLOW_EMBEDDING") == "true"

	fs.StringVar(&port, "p", port, "port")
	fs.DurationVar(&timeout, "t", timeout, "timeout")
}

func (c webCommand) Run(db *sql.DB, matcher *category.Matcher, logger *logger.Logger) error {
	handler, _ := router.New(db, matcher, logger)
	logger.Info("Starting web server", "url", fmt.Sprintf("http://localhost:%s", port))

	if !allowEmbedding {
		handler = xFrameDenyHeaderMiddleware(handler)
	}

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

func xFrameDenyHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
