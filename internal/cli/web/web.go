package web

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
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
			log.Println("[WARN] failed to parse custom timeout using default timout 5s")
		}
		timeout = duration
	} else {
		timeout = defaultTimeout
	}

	allowEmbedding = os.Getenv("EXPENSETRACE_ALLOW_EMBEDDING") == "true"

	fs.StringVar(&port, "p", port, "port")
	fs.DurationVar(&timeout, "t", timeout, "timeout")
}

func (c webCommand) Run(db *sql.DB, matcher *category.Matcher) error {
	handler, _ := router.New(db, matcher)
	log.Printf("Open report on http://localhost:%s\n", port)

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
			log.Fatalf("[ERROR] unexpected error %s", err.Error())
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan

	log.Println("Received signal, shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := server.Shutdown(ctx)
	if err != nil {
		log.Printf("[ERROR] issue shuting down server: %s\n", err.Error())
	}
	return nil
}

func xFrameDenyHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
