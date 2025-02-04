package report

import (
	"fmt"
	"sort"
	"time"

	"golang.org/x/exp/maps"

	pkgCategory "github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/expense"
)

type Category struct {
	Name     string
	Amount   int64
	Expenses []expense.Expense
}

type Report struct {
	Title                 string
	Spending              int64
	Income                int64
	Savings               int64
	SavingsPercentage     float32
	EarningsPerDay        int64
	AverageSpendingPerDay int64
	Categories            []Category
	Duplicates            []string
	Verbose               bool
}

func Generate(startDate, endDate time.Time, expenses []expense.Expense, reportType string) Report {
	var report Report

	var income int64
	var spending int64
	categories := make(map[string]Category)
	seen := map[string]bool{}
	duplicates := []string{}

	for _, ex := range expenses {
		if ex.Category == pkgCategory.Exclude {
			continue
		}

		_, ok := seen[ex.Description]
		if !ok {
			seen[ex.Description] = true
		} else {
			duplicates = append(duplicates, ex.Description)
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

	report.Income = income
	report.Spending = spending
	savings := income - spending
	report.Savings = savings
	report.SavingsPercentage = (float32(savings) / float32(income)) * 100
	numberOfDaysPerMonth := calendarDays(startDate, endDate)
	report.AverageSpendingPerDay = spending / int64(numberOfDaysPerMonth)
	report.EarningsPerDay = income / int64(numberOfDaysPerMonth)
	report.Duplicates = duplicates

	categoriesSlice := maps.Values(categories)

	sort.Slice(categoriesSlice, func(i, j int) bool {
		return categoriesSlice[i].Amount > categoriesSlice[j].Amount
	})

	report.Categories = categoriesSlice

	if reportType == "monthly" {
		report.Title = fmt.Sprintf("%s %d", startDate.Month().String(), startDate.Year())
	} else {
		report.Title = fmt.Sprintf("%d", startDate.Year())
	}

	return report
}

func addExpenseToCategory(categories map[string]Category, ex expense.Expense) {
	categoryName := expeseCategory(ex)

	c, ok := categories[categoryName]
	if ok {
		if ex.Type == expense.ChargeType {
			c.Amount -= ex.Amount
		} else {
			c.Amount += ex.Amount
		}
		c.Expenses = append(c.Expenses, ex)
		categories[categoryName] = c
	} else {
		var amount int64
		if ex.Type == expense.ChargeType {
			amount = -(ex.Amount)
		} else {
			amount = ex.Amount
		}
		c := Category{
			Amount: amount,
			Name:   categoryName,
			Expenses: []expense.Expense{
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

// calendarDays returns the calendar difference between times (t2 - t1) as days.
func calendarDays(t1, t2 time.Time) int {
	y, m, d := t2.Date()
	u2 := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	y, m, d = t1.In(t2.Location()).Date()
	u1 := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	days := u2.Sub(u1) / (24 * time.Hour)
	return int(days)
}
