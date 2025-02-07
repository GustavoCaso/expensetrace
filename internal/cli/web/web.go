package web

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
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
	parseTemplates()

	db, err := expenseDB.GetOrCreateExpenseDB(conf.DB)
	if err != nil {
		log.Fatalf("Unable to get expenses DB: %s", err.Error())
		os.Exit(1)
	}
	router := newRouter(db, conf)
	log.Printf("Open report on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), router))
}

func newRouter(db *sql.DB, conf *config.Config) *http.ServeMux {
	r := &http.ServeMux{}
	// Routes
	r.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		homeHandler(db, w, r)
	})

	r.HandleFunc("GET /import", func(w http.ResponseWriter, _ *http.Request) {
		err := importTempl.Execute(w, nil)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	r.HandleFunc("POST /import", func(w http.ResponseWriter, r *http.Request) {
		importHanlder(db, conf, w, r)
	})

	return r
}

type homeData struct {
	Report report.Report
	Error  error
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
		fmt.Fprint(w, errorMessage)
		return
	}

	fmt.Fprint(w, "Imported")
}

func homeHandler(db *sql.DB, w http.ResponseWriter, _ *http.Request) {
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

	err = indexTempl.Execute(w, data)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
