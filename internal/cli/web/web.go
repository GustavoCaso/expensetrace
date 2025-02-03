package web

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/config"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/expense"
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

func (c webCommand) Run(conf *config.Config) {
	db, err := expenseDB.GetOrCreateExpenseDB(conf.DB)
	if err != nil {
		log.Fatalf("Unable to get expenses DB: %s", err.Error())
		os.Exit(1)
	}
	router := newRouter(db)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), router.r))
}

type router struct {
	db *sql.DB
	r  *http.ServeMux
}

func newRouter(db *sql.DB) router {
	r := &http.ServeMux{}
	// Routes
	r.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		homeHandler(db, w, r)
	})

	return router{
		db: db,
		r:  r,
	}
}

func homeHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.New("home").Parse(`
		<!DOCTYPE html>
		<html>
			<head>
				<title>Expense Tracker</title>
				<script src="https://unpkg.com/htmx.org@1.6.1"></script>
			</head>
			<body>
				<h1>Expense Tracker</h1>

				<hr>
				
				{{ if eq .Error nil }}
					<!-- List of Expenses -->
					<h2>Expenses</h2>
					<ul id="expenses">
						{{range .Expenses}}
							<li>{{.Amount}} - {{.Category}} -- {{.Description}}</li>
						{{end}}
					</ul>
				{{ else }}
				 	<h2>There was an error: {{.Error}}</h2>
				{{ end }}
			</body>
		</html>
	`)

	// Fetch expenses from last month
	now := time.Now()

	lastMonth := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
	firstDay, lastDay := getMonthDates(lastMonth)

	expenses, err := expenseDB.GetExpensesFromDateRange(db, firstDay, lastDay)

	data := struct {
		Expenses []expense.Expense
		Error    error
	}{
		Expenses: expenses,
		Error:    err,
	}

	tmpl.Execute(w, data)
}

func getMonthDates(today time.Time) (time.Time, time.Time) {
	currentLocation := today.Location()

	firstOfMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, currentLocation)
	lastOfMonth := firstOfMonth.AddDate(0, 1, 0).Add(time.Nanosecond * -1)

	return firstOfMonth, lastOfMonth
}
