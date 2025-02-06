package web

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/config"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	importUtil "github.com/GustavoCaso/expensetrace/internal/import"
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
	router := newRouter(db, conf)
	log.Printf("Open report on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), router.r))
}

type router struct {
	db   *sql.DB
	conf *config.Config
	r    *http.ServeMux
}

func newRouter(db *sql.DB, conf *config.Config) router {
	r := &http.ServeMux{}
	// Routes
	r.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		homeHandler(db, w, r)
	})

	r.HandleFunc("POST /import", func(w http.ResponseWriter, r *http.Request) {
		importHanlder(db, conf, w, r)
	})

	return router{
		db:   db,
		r:    r,
		conf: conf,
	}
}

type homeData struct {
	Report report.Report
	Error  error
}

var templateFuncs = template.FuncMap{
	"formatMoney": util.FormatMoney,
}

func importHanlder(db *sql.DB, conf *config.Config, w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)

	file, header, err := r.FormFile("file")

	if err != nil {
		var errorMessage string
		if err == http.ErrMissingFile {
			errorMessage = "No file submitted"
		} else {
			errorMessage = "Error retrieving the file"
		}
		w.WriteHeader(400)
		fmt.Fprint(w, errorMessage)
		return
	}
	defer file.Close()

	// Copy the file data to my buffer
	var buf bytes.Buffer
	io.Copy(&buf, file)
	log.Printf("Importing File name %s. Size %dKB\n", header.Filename, buf.Len())
	categoryMatcher := category.New(conf.Categories)
	errors := importUtil.Import(header.Filename, &buf, db, categoryMatcher)

	if len(errors) > 0 {
		errorStrings := make([]string, len(errors))
		for i, err := range errors {
			errorStrings[i] = err.Error()
		}
		errorMessage := strings.Join(errorStrings, "\n")
		log.Printf("Errors importing file: %s. %s", header.Filename, errorMessage)
		w.WriteHeader(400)
		fmt.Fprint(w, errorMessage)
		return
	}

	w.WriteHeader(201)
	fmt.Fprint(w, "Imported")
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
				
				<form id='form'>
        	<input type='file' name='file' required>
        	 <button 
						hx-post="/import" 
						hx-encoding="multipart/form-data" 
						hx-target="#form-results" 
						type="submit">Import</button>
    		</form>
				<div id='form-results'>

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
