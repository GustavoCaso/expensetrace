package report

import (
	"database/sql"
	"embed"
	"flag"
	"io"
	"log"
	"os"
	"path"
	"text/template"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/config"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/expense"
	internalReport "github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

// content holds our static content.
//
//go:embed templates/*
var content embed.FS

type reportCommand struct {
}

func NewCommand() cli.Command {
	return reportCommand{}
}

func (c reportCommand) Description() string {
	return "Displays the expenses information for selected date ranges"
}

var month int
var year int
var verbose bool

func (c reportCommand) SetFlags(fs *flag.FlagSet) {
	fs.IntVar(&month, "month", -1, "what month to use for generating report")
	fs.IntVar(&year, "year", -1, "what year to use for generating report")
	fs.BoolVar(&verbose, "v", false, "show verbose report output")
}

func (c reportCommand) Run(conf *config.Config, db *sql.DB) {
	defer db.Close()

	now := time.Now()
	var startDate, endDate time.Time
	var reportType string

	if month == -1 && year == -1 {
		// Using default values we setup current year and previous month
		// and display the monthly report
		currentMonth := now.Month()
		currentYear := now.Year()
		reportType = "monthly"
		startDate, endDate = util.GetMonthDates(int(currentMonth-1), currentYear)
	} else {
		if month > 0 {
			reportType = "monthly"
			startDate, endDate = util.GetMonthDates(month, year)
		} else if month == 0 && year > 0 {
			reportType = "yearly"
			startDate, endDate = util.GetYearDates(year)
		}
	}

	expenses, err := getExpenses(startDate, endDate, db)
	if err != nil {
		log.Fatalf("Unable to fetch expenses: %v", err.Error())
	}
	r := internalReport.Generate(startDate, endDate, expenses, reportType)
	r.Verbose = verbose

	err = renderTemplate(os.Stdout, "report.tmpl", r)

	if err != nil {
		log.Fatalf("Unable to render report: %v", err.Error())
	}

	os.Exit(0)
}

func getExpenses(startDate, endDate time.Time, db *sql.DB) ([]expense.Expense, error) {
	expenses, err := expenseDB.GetExpensesFromDateRange(db, startDate, endDate)
	if err != nil {
		return []expense.Expense{}, err
	}

	return expenses, nil
}

var templateFuncs = template.FuncMap{
	"formatMoney": util.FormatMoney,
	"colorOutput": util.ColorOutput,
}

func renderTemplate(out io.Writer, templateName string, value interface{}) error {
	tmpl, err := content.ReadFile(path.Join("templates", templateName))
	if err != nil {
		return err
	}
	t := template.Must(template.New(templateName).Funcs(templateFuncs).Parse(string(tmpl)))
	err = t.Execute(out, value)
	if err != nil {
		return err
	}

	return nil
}
