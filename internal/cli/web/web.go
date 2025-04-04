package web

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

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

func (c webCommand) SetFlags(fs *flag.FlagSet) {
	port = os.Getenv("EXPENSETRACE_PORT")
	if port == "" {
		port = "8080"
	}

	fs.StringVar(&port, "p", port, "port")
}

func (c webCommand) Run(db *sql.DB, matcher *category.Matcher) error {
	handler, _ := router.New(db, matcher)
	log.Printf("Open report on http://localhost:%s\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%s", port), handler)
}
