package main

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

	pkgCategory "github.com/GustavoCaso/expensetrace/pkg/category"
	"github.com/GustavoCaso/expensetrace/pkg/config"
	expenseDB "github.com/GustavoCaso/expensetrace/pkg/db"
	"github.com/GustavoCaso/expensetrace/pkg/expense"
	"github.com/GustavoCaso/expensetrace/pkg/util"
	"github.com/fatih/color"
)

var month = flag.Int("month", 0, "what month to use for calculating the report")
var verbose = flag.Bool("v", false, "show verbose report output")

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
	Year                  int
	Month                 string
	Spending              int64
	Income                int64
	Savings               int64
	SavingsPercentage     float32
	EarningsPerDay        int64
	AverageSpendingPerDay int64
	Categories            []category
	Verbose               bool
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "c", "expense.toml", "Configuration file")
	flag.Parse()

	conf, err := config.Parse(configPath)

	if err != nil {
		log.Fatalf("Unable to parse the configuration: %s", err.Error())
	}

	if *month == 0 {
		log.Fatal("You must provide the moth you want to generate the report")
		os.Exit(1)
	}

	goMonth := time.Month(*month)

	today := time.Now()
	currentLocation := today.Location()

	firstOfMonth := time.Date(today.Year(), goMonth, 1, 0, 0, 0, 0, currentLocation)
	lastOfMonth := firstOfMonth.AddDate(0, 1, 0).Add(time.Nanosecond * -1)

	db, err := expenseDB.GetOrCreateExpenseDB(conf.DB)
	if err != nil {
		log.Fatalf("Unable to get expenses DB: %s", err.Error())
		os.Exit(1)
	}

	defer db.Close()
	expenses, err := expenseDB.GetExpensesFromDateRange(db, firstOfMonth, lastOfMonth)
	if err != nil {
		log.Fatalf("Unable to get expenses: %s", err.Error())
		os.Exit(1)
	}

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
	report.Verbose = *verbose
	report.Year = today.Year()
	report.Month = goMonth.String()
	report.Income = income
	report.Spending = spending
	savings := income - spending
	report.Savings = savings
	report.SavingsPercentage = (float32(savings) / float32(income)) * 100

	numberOfDaysPerMonth := calendarDays(lastOfMonth, firstOfMonth)
	report.AverageSpendingPerDay = spending / int64(numberOfDaysPerMonth)
	report.EarningsPerDay = income / int64(numberOfDaysPerMonth)

	categoriesSlice := maps.Values(categories)

	sort.Slice(categoriesSlice, func(i, j int) bool {
		return categoriesSlice[i].amount > categoriesSlice[j].amount
	})

	report.Categories = categoriesSlice

	err = renderTemplate(os.Stdout, "report.tmpl", report)
	if err != nil {
		log.Fatalf("Error rendering the report: %s", err.Error())
	}

	os.Exit(0)
}

func addExpenseToCategory(categories map[string]category, ex expense.Expense) {
	c, ok := categories[ex.Category]
	if ok {
		c.amount += ex.Amount
		c.expenses = append(c.expenses, ex)
		categories[ex.Category] = c
	} else {
		var cName string
		if ex.Category == "" {
			cName = "uncategorized"
		} else {
			cName = ex.Category
		}
		c := category{
			amount:       ex.Amount,
			name:         cName,
			categoryType: ex.Type,
			expenses: []expense.Expense{
				ex,
			},
		}
		categories[ex.Category] = c
	}
}

// calendarDays returns the calendar difference between times (t2 - t1) as days.
func calendarDays(t2, t1 time.Time) int {
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
