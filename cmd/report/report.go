package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
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
	amount       float32
	categoryType expense.ExpenseType
}

func (c category) Display() string {
	var sign string
	if c.categoryType == expense.IncomeType {
		sign = "+"
	} else {
		sign = "-"
	}

	return fmt.Sprintf("%s %s%.2f", c.name, sign, c.amount)
}

type Report struct {
	Spending              float32
	Income                float32
	Savings               float32
	EarningsPerday        float32
	AverageSpendingPerDay float32
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
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

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

	var income float32
	var spending float32
	categories := make(map[string]category)

	for _, ex := range expenses {
		if ex.Category == pkgCategory.Exclude {
			continue
		}

		switch ex.Type {
		case expense.ChargeType:
			value := fmt.Sprintf("%d.%d", ex.Amount, ex.Decimal)
			v, err := strconv.ParseFloat(value, 32)
			if err != nil {
				log.Fatalf("Unable to parse numbre: %s", err.Error())
				os.Exit(1)
			}
			spending += float32(v)
			addCategory(categories, ex, v)
		case expense.IncomeType:
			value := fmt.Sprintf("%d.%d", ex.Amount, ex.Decimal)
			v, err := strconv.ParseFloat(value, 32)
			if err != nil {
				log.Fatalf("Unable to parse numbre: %s", err.Error())
				os.Exit(1)
			}
			income += float32(v)
			addCategory(categories, ex, v)
		}
	}

	report.Income = income
	report.Spending = spending
	report.Savings = income - spending

	numberOfDaysPerMonth := calendarDays(lastOfMonth, firstOfMonth)
	report.AverageSpendingPerDay = spending / float32(numberOfDaysPerMonth)
	report.EarningsPerday = income / float32(numberOfDaysPerMonth)

	categoriesSlice := maps.Values(categories)

	sort.Slice(categoriesSlice, func(i, j int) bool {
		return categoriesSlice[i].name < categoriesSlice[j].name
	})

	report.Categories = categoriesSlice

	tmpl, tmplErr := content.ReadFile(path.Join("templates", "report.tmpl"))
	if tmplErr != nil {
		log.Fatalf("Unable to parse template: %s", tmplErr.Error())
		os.Exit(1)
	}
	t := template.Must(template.New("report").Parse(string(tmpl)))
	err = t.Execute(os.Stdout, report)
	if err != nil {
		log.Fatalf("Error rendering the report: %s", err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func addCategory(categories map[string]category, ex expense.Expense, v float64) {
	c, ok := categories[ex.Category]
	if ok {
		c.amount += float32(v)
		categories[ex.Category] = c
	} else {
		var cName string
		if ex.Category == "" {
			cName = "uncategorized"
		} else {
			cName = ex.Category
		}
		c := category{
			amount:       float32(v),
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
