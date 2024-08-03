package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path"
	"sort"
	"time"

	"golang.org/x/exp/maps"

	pkgCategory "github.com/GustavoCaso/sandbox/go/moneyTracker/pkg/category"
	expenseDB "github.com/GustavoCaso/sandbox/go/moneyTracker/pkg/db"
	"github.com/GustavoCaso/sandbox/go/moneyTracker/pkg/expense"
)

var month = flag.Int("month", 0, "what month to use for calculating the report")

// content holds our static content.
//
//go:embed templates/*
var content embed.FS

type category struct {
	name         string
	amount       int64
	categoryType expense.ExpenseType
}

func (c category) Display() string {
	value := c.amount

	if c.categoryType == expense.ChargeType {
		value = -(value)
	}

	return fmt.Sprintf("%s: %s€", c.name, formatMoney(value, ".", ","))
}

type Report struct {
	Spending              int64
	Income                int64
	Savings               int64
	EarningsPerDay        int64
	AverageSpendingPerDay int64
	Categories            []category
}

func main() {
	flag.Parse()

	if *month == 0 {
		log.Fatal("You must provide the moth you want to generate the report")
		os.Exit(1)
	}

	today := time.Now()
	currentLocation := today.Location()

	firstOfMonth := time.Date(today.Year(), time.Month(*month), 1, 0, 0, 0, 0, currentLocation)
	lastOfMonth := firstOfMonth.AddDate(0, 1, 0).Add(time.Nanosecond * -1)

	db, err := expenseDB.GetOrCreateExpenseDB()
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
			addCategory(categories, ex, ex.Amount)
		case expense.IncomeType:
			income += ex.Amount
			addCategory(categories, ex, ex.Amount)
		}
	}

	report.Income = income
	report.Spending = spending
	report.Savings = income - spending

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

func addCategory(categories map[string]category, ex expense.Expense, v int64) {
	c, ok := categories[ex.Category]
	if ok {
		c.amount += v
		categories[ex.Category] = c
	} else {
		var cName string
		if ex.Category == "" {
			cName = "uncategorized"
		} else {
			cName = ex.Category
		}
		c := category{
			amount:       v,
			name:         cName,
			categoryType: ex.Type,
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

func formatMoney(value int64, thousand, decimal string) string {
	var result string
	var isNegative bool

	if value < 0 {
		value = value * -1
		isNegative = true
	}

	// apply the decimal separator
	result = fmt.Sprintf("%s%02d%s", decimal, value%100, result)
	value /= 100

	// for each 3 dígits put a dot "."
	for value >= 1000 {
		result = fmt.Sprintf("%s%03d%s", thousand, value%1000, result)
		value /= 1000
	}

	if isNegative {
		return fmt.Sprintf("-%d%s", value, result)
	}

	return fmt.Sprintf("%d%s", value, result)
}

var templateFuncs = template.FuncMap{
	"formatMoney": formatMoney,
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
