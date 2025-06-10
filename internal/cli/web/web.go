package web

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
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
var timeout int
var allowEmbedding = false

const (
	defaultPort    = "8080"
	defaultTimeout = 3
)

func (c webCommand) SetFlags(fs *flag.FlagSet) {
	port = os.Getenv("EXPENSETRACE_PORT")
	if port == "" {
		port = defaultPort
	}

	allowEmbedding = os.Getenv("EXPENSETRACE_ALLOW_EMBEDDING") == "true"

	fs.StringVar(&port, "p", port, "port")
	fs.IntVar(&timeout, "t", defaultTimeout, "timeout")
}

func (c webCommand) Run(db *sql.DB, matcher *category.Matcher) error {
	handler, _ := router.New(db, matcher)
	log.Printf("Open report on http://localhost:%s\n", port)

	if !allowEmbedding {
		handler = xFrameDenyHeaderMiddleware(handler)
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		ReadHeaderTimeout: time.Duration(timeout) * time.Second,
		Handler:           handler,
	}
	return server.ListenAndServe()
}

func xFrameDenyHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}
