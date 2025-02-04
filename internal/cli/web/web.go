package web

import (
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/config"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/util"
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
	log.Printf("Open report on http://localhost:%s\n", port)
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

type homeData struct {
	Report report.Report
	Error  error
}

var templateFuncs = template.FuncMap{
	"formatMoney": util.FormatMoney,
}

func homeHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("home").Funcs(templateFuncs).Parse(`
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
					{{with .Report }}
						<h2>{{.Title}}</h2>
						<ul id="summary">
							<li style="color:green;"><b>Income: {{formatMoney .Income "." ","}}</b>€</li>
							<li style="color:crimson;"><u>Spending:  {{formatMoney .Spending "." ","}}€</u></li>

							{{if gt .Savings 0}}
								<li style="color:green;">Savings: {{formatMoney .Savings "." ","}}€ <b>{{printf "%.2f%%" .SavingsPercentage}}</b></li>
							{{else}}
								<li style="color:crimson;"> Savings: {{formatMoney .Savings "." ","}}€ <u>{{printf "%.2f%%" .SavingsPercentage}}</u></li>
							{{end}}
						</ul>
					
						<p> Breakdown by category </p>
						<ul id="categories">
							{{range $category := .Categories}}
								{{if gt $category.Amount 0}}
									<li>{{$category.Name}}: <span style="color:green;"><b>{{formatMoney $category.Amount "." ","}}</b></span></li>
								{{else}}
									<li>{{$category.Name}}: <span style="color:crimson;"><u>{{formatMoney $category.Amount "." ","}}</u></span></li>
								{{end}}
							{{end}}
						</ul>
					{{end}}
				{{ else }}
				 	<h2>There was an error: {{.Error}}</h2>
				{{ end }}
			</body>
		</html>
	`)

	if err != nil {
		log.Fatalf("error parsing home template %v", err)
	}

	// Fetch expenses from last month
	now := time.Now()

	firstDay, lastDay := util.GetMonthDates(int(now.Month()-1), now.Year())

	var data homeData
	expenses, err := expenseDB.GetExpensesFromDateRange(db, firstDay, lastDay)

	if err != nil {
		data = homeData{
			Error: err,
		}
	} else {
		result := report.Generate(firstDay, lastDay, expenses, "monthly")

		data = homeData{
			Report: result,
			Error:  nil,
		}
	}

	tmpl.Execute(w, data)
}
