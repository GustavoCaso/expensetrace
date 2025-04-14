package report

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"time"

	pkgCategory "github.com/GustavoCaso/expensetrace/internal/category"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
)

type Category struct {
	Name              string
	Amount            int64
	Expenses          []*expenseDB.Expense
	PercentageOfTotal float64
	LastTransaction   time.Time
	AvgAmount         int64
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

const (
	percentageOfTotal = 100
)

func Generate(startDate, endDate time.Time, expenses []*expenseDB.Expense, reportType string) Report {
	var report Report

	categories, duplicates, income, spending := Categories(expenses)

	report.Income = income
	report.Spending = spending
	report.StartDate = startDate
	report.EndDate = endDate
	savings := income - (spending)*-1
	report.Savings = savings
	report.SavingsPercentage = (float32(savings) / float32(income)) * percentageOfTotal
	numberOfDaysPerMonth := calendarDays(startDate, endDate)
	report.AverageSpendingPerDay = (spending) * -1 / int64(numberOfDaysPerMonth)
	report.EarningsPerDay = income / int64(numberOfDaysPerMonth)
	report.Duplicates = duplicates

	categoriesSlice := slices.Collect(maps.Values(categories))

	sort.Slice(categoriesSlice, func(i, j int) bool {
		return categoriesSlice[i].Amount > categoriesSlice[j].Amount
	})

	report.Categories = categoriesSlice

	if reportType == "monthly" {
		report.Title = fmt.Sprintf("%s %d", startDate.Month().String(), startDate.Year())
	} else {
		report.Title = strconv.Itoa(startDate.Year())
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
		case expenseDB.IncomeType:
			income += ex.Amount
		}

		addExpenseToCategory(categories, ex)
	}

	for key, category := range categories {
		// Calculate percentage of total
		if category.Amount < 0 && spending < 0 {
			// For expense categories
			category.PercentageOfTotal = float64(category.Amount*-percentageOfTotal) / float64(spending*-1)
		} else if category.Amount > 0 && income > 0 {
			// For income categories
			category.PercentageOfTotal = float64(category.Amount*percentageOfTotal) / float64(income)
		}

		// Find most recent transaction
		if len(category.Expenses) > 0 {
			category.LastTransaction = category.Expenses[0].Date
			// Calculate average amount
			total := int64(0)
			for _, exp := range category.Expenses {
				total += exp.Amount

				if exp.Date.After(category.LastTransaction) {
					category.LastTransaction = exp.Date
				}
			}
			category.AvgAmount = total / int64(len(category.Expenses))
		}

		categories[key] = category
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
		cat := Category{
			Amount: amount,
			Name:   categoryName,
			Expenses: []*expenseDB.Expense{
				ex,
			},
		}
		categories[categoryName] = cat
	}
}

func expeseCategory(ex *expenseDB.Expense) string {
	category, err := ex.Category()
	if err != nil {
		fmt.Printf("error fetching the category: %+v\n", err.Error())
	}

	if category == "" {
		if ex.Type == expenseDB.IncomeType {
			return "uncategorized income"
		}

		return "uncategorized charge"
	}
	return category
}

const (
	hoursInDay = 24
)

// calendarDays returns the calendar difference between times (t2 - t1) as days.
func calendarDays(t1, t2 time.Time) int {
	y, m, d := t2.Date()
	u2 := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	y, m, d = t1.In(t2.Location()).Date()
	u1 := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	days := u2.Sub(u1) / (hoursInDay * time.Hour)
	return int(days)
}
