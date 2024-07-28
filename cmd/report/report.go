package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"path"
	"strconv"
	"time"

	expenseDB "github.com/GustavoCaso/sandbox/go/moneyTracker/pkg/db"
	"github.com/GustavoCaso/sandbox/go/moneyTracker/pkg/expense"
)

var month = flag.Int("month", 0, "what month to use for calculating the report")

// content holds our static content.
//
//go:embed templates/*
var content embed.FS

type Report struct {
	Spending   float32
	Income     float32
	Savings    float32
	Categories map[string]float32
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
	categories := make(map[string]float32)

	for _, ex := range expenses {
		switch ex.Type {
		case expense.ChargeType:
			value := fmt.Sprintf("%d.%d", ex.Amount, ex.Decimal)
			v, err := strconv.ParseFloat(value, 32)
			if err != nil {
				log.Fatalf("Unable to parse numbre: %s", err.Error())
				os.Exit(1)
			}
			spending += float32(v)
			categories[ex.Category] += float32(v)
		case expense.IncomeType:
			value := fmt.Sprintf("%d.%d", ex.Amount, ex.Decimal)
			v, err := strconv.ParseFloat(value, 32)
			if err != nil {
				log.Fatalf("Unable to parse numbre: %s", err.Error())
				os.Exit(1)
			}
			income += float32(v)
			categories[ex.Category] += float32(v)
		}
	}

	report.Income = income
	report.Spending = spending
	report.Savings = income - spending
	report.Categories = categories

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
