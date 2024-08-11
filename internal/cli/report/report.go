package report

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"sort"
	"text/template"
	"time"

	"golang.org/x/exp/maps"

	pkgCategory "github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/config"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/expense"
	"github.com/GustavoCaso/expensetrace/internal/util"
	"github.com/fatih/color"
)

// content holds our static content.
//
//go:embed templates/*
var content embed.FS

type category struct {
	name         string
	amount       int64
	categoryType expense.ExpenseType
	expenses     []expense.Expense
}

func (c category) Display(verbose bool) string {
	value := c.amount

	if c.categoryType == expense.ChargeType {
		value = -(value)
	}

	var buffer = bytes.Buffer{}

	if value < 0 {
		buffer.WriteString(fmt.Sprintf("%s: %s€", c.name, colorOutput(util.FormatMoney(value, ".", ","), "red", "underline")))
	} else {
		buffer.WriteString(fmt.Sprintf("%s: %s€", c.name, colorOutput(util.FormatMoney(value, ".", ","), "green", "bold")))
	}

	if verbose {
		buffer.WriteString("\n")
		for _, ex := range c.expenses {
			buffer.WriteString(fmt.Sprintf("%s %s %s€\n", ex.Date.Format("2006-01-02"), ex.Description, util.FormatMoney(ex.Amount, ".", ",")))
		}
	}

	return buffer.String()
}

type Report struct {
	Title                 string
	Spending              int64
	Income                int64
	Savings               int64
	SavingsPercentage     float32
	EarningsPerDay        int64
	AverageSpendingPerDay int64
	Categories            []category
	Verbose               bool
}

type reportCommand struct {
}

func NewCommand() cli.Command {
	return reportCommand{}
}

var month int
var year int
var verbose bool

func (c reportCommand) SetFlags(fs *flag.FlagSet) {
	fs.IntVar(&month, "month", 0, "what month to use for generating report")
	fs.IntVar(&year, "year", 0, "what year to use for generating report")
	fs.BoolVar(&verbose, "v", false, "show verbose report output")
}

func (c reportCommand) Run(conf *config.Config) {
	if month == 0 && year == 0 {
		log.Fatal("You must provide either the month or year to generate the report")
		os.Exit(1)
	}

	var reportType string
	var startDate, endDate time.Time
	if month > 0 {
		reportType = "monthly"
		startDate, endDate = getMonthDates(month, year)
	} else if month == 0 && year > 0 {
		reportType = "yearly"
		startDate, endDate = getYearDates(year)
	}

	expenses, err := getExpenses(startDate, endDate, conf)
	if err != nil {
		log.Fatalf("Unable to fetch expenses: %v", err.Error())
	}
	err = generateReport(startDate, endDate, expenses, reportType)
	if err != nil {
		log.Fatalf("Unable to render report: %v", err.Error())
	}

	os.Exit(0)
}

func generateReport(startDate, endDate time.Time, expenses []expense.Expense, reportType string) error {
	var report Report

	var income int64
	var spending int64
	categories := make(map[string]category)

	for _, ex := range expenses {
		if ex.Category == pkgCategory.Exclude {
			continue
		}

		switch ex.Type {
		case expense.ChargeType:
			spending += ex.Amount
			addExpenseToCategory(categories, ex)
		case expense.IncomeType:
			income += ex.Amount
			addExpenseToCategory(categories, ex)
		}
	}

	report.Verbose = verbose
	report.Income = income
	report.Spending = spending
	savings := income - spending
	report.Savings = savings
	report.SavingsPercentage = (float32(savings) / float32(income)) * 100
	numberOfDaysPerMonth := calendarDays(startDate, endDate)
	report.AverageSpendingPerDay = spending / int64(numberOfDaysPerMonth)
	report.EarningsPerDay = income / int64(numberOfDaysPerMonth)

	categoriesSlice := maps.Values(categories)

	sort.Slice(categoriesSlice, func(i, j int) bool {
		return categoriesSlice[i].amount > categoriesSlice[j].amount
	})

	report.Categories = categoriesSlice

	if reportType == "monthly" {
		report.Title = fmt.Sprintf("%s %d", startDate.Month().String(), startDate.Year())
	} else {
		report.Title = fmt.Sprintf("%d", startDate.Year())
	}

	err := renderTemplate(os.Stdout, "report.tmpl", report)
	if err != nil {
		return err
	}

	return nil
}

func addExpenseToCategory(categories map[string]category, ex expense.Expense) {
	categoryName := expeseCategory(ex)

	c, ok := categories[categoryName]
	if ok {
		c.amount += ex.Amount
		c.expenses = append(c.expenses, ex)
		categories[categoryName] = c
	} else {
		c := category{
			amount:       ex.Amount,
			name:         categoryName,
			categoryType: ex.Type,
			expenses: []expense.Expense{
				ex,
			},
		}
		categories[categoryName] = c
	}
}

func expeseCategory(ex expense.Expense) string {
	if ex.Category == "" {
		if ex.Type == expense.IncomeType {
			return "uncategorized income"
		} else {
			return "uncategorized charge"
		}
	}
	return ex.Category
}

func getMonthDates(month int, year int) (time.Time, time.Time) {
	goMonth := time.Month(month)

	today := time.Now()
	currentLocation := today.Location()

	var y int
	if year > 0 {
		y = year
	} else {
		y = today.Year()
	}

	firstOfMonth := time.Date(y, goMonth, 1, 0, 0, 0, 0, currentLocation)
	lastOfMonth := firstOfMonth.AddDate(0, 1, 0).Add(time.Nanosecond * -1)

	return firstOfMonth, lastOfMonth
}

func getYearDates(year int) (time.Time, time.Time) {
	today := time.Now()
	currentLocation := today.Location()

	firstOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, currentLocation)
	lastOfYear := time.Date(year, 12, 31, 0, 0, 0, 0, currentLocation)

	return firstOfYear, lastOfYear
}

func getExpenses(startDate, endDate time.Time, conf *config.Config) ([]expense.Expense, error) {
	db, err := expenseDB.GetOrCreateExpenseDB(conf.DB)
	if err != nil {
		return []expense.Expense{}, err
	}

	defer db.Close()
	expenses, err := expenseDB.GetExpensesFromDateRange(db, startDate, endDate)
	if err != nil {
		return []expense.Expense{}, err
	}

	return expenses, nil
}

// calendarDays returns the calendar difference between times (t2 - t1) as days.
func calendarDays(t1, t2 time.Time) int {
	y, m, d := t2.Date()
	u2 := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	y, m, d = t1.In(t2.Location()).Date()
	u1 := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	days := u2.Sub(u1) / (24 * time.Hour)
	return int(days)
}

var colorsOptions = map[string]color.Attribute{
	"red":       color.FgHiRed,
	"green":     color.FgGreen,
	"underline": color.Underline,
	"bold":      color.Bold,
	"bgRed":     color.BgRed,
	"bgGreen":   color.BgGreen,
}

func colorOutput(text string, colorOptions ...string) string {
	attributes := []color.Attribute{}
	for _, option := range colorOptions {
		if o, ok := colorsOptions[option]; ok {
			attributes = append(attributes, o)
		}

	}
	c := color.New(attributes...)
	return c.Sprint(text)
}

var templateFuncs = template.FuncMap{
	"formatMoney": util.FormatMoney,
	"colorOutput": colorOutput,
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
