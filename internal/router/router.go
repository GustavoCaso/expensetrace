package router

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/config"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/expense"
	importUtil "github.com/GustavoCaso/expensetrace/internal/import"
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

func New(db *sql.DB, conf *config.Config) *http.ServeMux {
	parseTemplates()

	r := &http.ServeMux{}
	// Routes
	r.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		homeHandler(db, w, r)
	})

	r.HandleFunc("GET /expenses", func(w http.ResponseWriter, _ *http.Request) {
		expensesHandler(db, w)
	})

	r.HandleFunc("GET /import", func(w http.ResponseWriter, _ *http.Request) {
		err := importTempl.Execute(w, nil)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	r.HandleFunc("POST /search", func(w http.ResponseWriter, r *http.Request) {
		searchHandler(db, w, r)
	})

	r.HandleFunc("POST /import", func(w http.ResponseWriter, r *http.Request) {
		importHandler(db, conf, w, r)
	})

	return r
}

type homeData struct {
	Report report.Report
	Error  error
}

func importHandler(db *sql.DB, conf *config.Config, w http.ResponseWriter, r *http.Request) {
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

func searchHandler(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	query := r.FormValue("q")

	if query == "" {
		fmt.Fprint(w, "You must provide a search criteria")
		return
	}

	expenses, err := expenseDB.SearchExpenses(db, query)
	if err != nil {
		panic(err)
	}

	sort.Slice(expenses, func(i, j int) bool {
		return expenses[i].Date.Unix() < expenses[j].Date.Unix()
	})

	categories, _, _, _ := report.Categories(expenses)

	data := struct {
		Categories map[string]report.Category
		Query      string
	}{
		Categories: categories,
		Query:      query,
	}

	err = searchResultsTempl.ExecuteTemplate(w, "searchResults.html", data)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

}

func expensesHandler(db *sql.DB, w http.ResponseWriter) {
	expenses, err := expenseDB.GetExpenses(db)
	if err != nil {
		data := struct {
			Error error
		}{
			Error: err,
		}
		err = expensesTempl.Execute(w, data)
		if err != nil {
			log.Print(err.Error())
			errorMessage := fmt.Sprintf("Internal Server Error: %v", err.Error())
			w.Write([]byte(errorMessage))
			return
		}
	}

	sort.Slice(expenses, func(i, j int) bool {
		return expenses[i].Date.Unix() > expenses[j].Date.Unix()
	})

	data := struct {
		Expenses []expense.Expense
		Error    error
	}{
		Expenses: expenses,
	}

	err = expensesTempl.Execute(w, data)
	if err != nil {
		log.Print(err.Error())
		errorMessage := fmt.Sprintf("Internal Server Error: %v", err.Error())
		w.Write([]byte(errorMessage))
	}
}
