package web

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"

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
	fs.StringVar(&port, "p", "8080", "port")
}

func (c webCommand) Run(db *sql.DB, matcher *category.Matcher) {
	defer db.Close()
	router := router.New(db, matcher)
	log.Printf("Open report on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), router))
}
