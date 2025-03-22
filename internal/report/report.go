package report

import (
	"fmt"
	"sort"
	"time"

	"golang.org/x/exp/maps"

	pkgCategory "github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
)

type Category struct {
	Name     string
	Amount   int64
	Expenses []*expenseDB.Expense
}

type Report struct {
	Title                 string
	Spending              int64
	Income                int64
	Savings               int64
	StartDate             time.Time
	EndDate               time.Time
	SavingsPercentage     float32
	EarningsPerDay        int64
	AverageSpendingPerDay int64
	Categories            []Category
	Duplicates            []string
	Verbose               bool
}

func Generate(startDate, endDate time.Time, expenses []*expenseDB.Expense, reportType string) Report {
	var report Report

	categories, duplicates, income, spending := Categories(expenses)

	report.Income = income
	report.Spending = spending
	report.StartDate = startDate
	report.EndDate = endDate
	savings := income - (spending)*-1
	report.Savings = savings
	report.SavingsPercentage = (float32(savings) / float32(income)) * 100
	numberOfDaysPerMonth := calendarDays(startDate, endDate)
	report.AverageSpendingPerDay = (spending) * -1 / int64(numberOfDaysPerMonth)
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

func Categories(expenses []*expenseDB.Expense) (map[string]Category, []string, int64, int64) {
	var income int64
	var spending int64
	categories := make(map[string]Category)
	seen := map[string]bool{}
	duplicates := []string{}

	for _, ex := range expenses {
		category, err := ex.Category()
		if err != nil {
			fmt.Printf("error fetching category: %+v\n", err.Error())
		}

		if category == pkgCategory.Exclude {
			continue
		}

		_, ok := seen[ex.Description]
		if !ok {
			seen[ex.Description] = true
		} else {
			duplicates = append(duplicates, ex.Description)
		}

		switch ex.Type {
		case expenseDB.ChargeType:
			spending += ex.Amount
			addExpenseToCategory(categories, ex)
		case expenseDB.IncomeType:
			income += ex.Amount
			addExpenseToCategory(categories, ex)
		}
	}

	return categories, duplicates, income, spending
}

func addExpenseToCategory(categories map[string]Category, ex *expenseDB.Expense) {
	categoryName := expeseCategory(ex)

	c, ok := categories[categoryName]
	if ok {
		c.Amount += ex.Amount
		c.Expenses = append(c.Expenses, ex)
		categories[categoryName] = c
	} else {
		amount := ex.Amount
		c := Category{
			Amount: amount,
			Name:   categoryName,
			Expenses: []*expenseDB.Expense{
				ex,
			},
		}
		categories[categoryName] = c
	}
}

func expeseCategory(ex *expenseDB.Expense) string {
	category, err := ex.Category()
	if err != nil {
		fmt.Printf("error fetching the category: %+v\n", err.Error())
		return ""
	}

	if category == "" {
		if ex.Type == expenseDB.IncomeType {
			return "uncategorized income"
		} else {
			return "uncategorized charge"
		}
	}
	return category
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
