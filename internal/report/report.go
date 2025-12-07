package report

import (
	"context"
	"fmt"
	"maps"
	"math"
	"slices"
	"sort"
	"strconv"
	"time"

	pkgStorage "github.com/GustavoCaso/expensetrace/internal/storage"
)

type BudgetStatus string

const (
	BudgetStatusUnder    BudgetStatus = "under"
	BudgetStatusNear     BudgetStatus = "near"
	BudgetStatusOver     BudgetStatus = "over"
	BudgetStatusNoBudget BudgetStatus = "no_budget"
)

const (
	budgetUnder = 80
	budgetFull  = 100
)

type BudgetInfo struct {
	Amount         int64        // Budget amount in cents (0 if no budget)
	Spent          int64        // Amount spent (positive)
	Remaining      int64        // Remaining budget (can be negative)
	PercentageUsed float64      // Percentage of budget used
	Status         BudgetStatus // Color coding status
}

type Category struct {
	Name              string
	Amount            int64
	Expenses          []pkgStorage.Expense
	PercentageOfTotal float64
	LastTransaction   time.Time
	AvgAmount         int64
	Budget            BudgetInfo
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

func Generate(
	ctx context.Context,
	userID int64,
	startDate, endDate time.Time,
	storage pkgStorage.Storage,
	expenses []pkgStorage.Expense,
	reportType string,
) (Report, error) {
	var report Report

	categories, duplicates, income, spending, err := Categories(ctx, userID, storage, expenses)

	if err != nil {
		return report, err
	}

	report.Income = income
	report.Spending = spending
	report.StartDate = startDate
	report.EndDate = endDate
	savings := income - (spending)*-1
	report.Savings = savings
	savingsPercentage := (float32(savings) / float32(income)) * percentageOfTotal
	if math.IsNaN(float64(savingsPercentage)) || math.IsInf(float64(savingsPercentage), 0) {
		report.SavingsPercentage = 0
	} else {
		report.SavingsPercentage = savingsPercentage
	}
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

	return report, nil
}

func Categories(
	ctx context.Context,
	userID int64,
	storage pkgStorage.Storage,
	expenses []pkgStorage.Expense,
) (map[string]Category, []string, int64, int64, error) {
	var income int64
	var spending int64
	categories := make(map[string]Category)
	seen := map[string]bool{}
	duplicates := []string{}
	// Track category budgets by category name
	categoryBudgets := make(map[string]int64)

	for _, ex := range expenses {
		categoryName := ""
		if ex.CategoryID() != nil {
			category, categoryError := storage.GetCategory(ctx, userID, *ex.CategoryID())

			if categoryError != nil {
				return categories, duplicates, income, spending, categoryError
			}

			if category.Name() == pkgStorage.ExcludeCategory {
				continue
			}
			categoryName = category.Name()
			// Store budget for this category
			categoryBudgets[categoryName] = category.MonthlyBudget()
		}

		_, ok := seen[ex.Description()]
		if !ok {
			seen[ex.Description()] = true
		} else {
			duplicates = append(duplicates, ex.Description())
		}

		switch ex.Type() {
		case pkgStorage.ChargeType:
			spending += ex.Amount()
		case pkgStorage.IncomeType:
			income += ex.Amount()
		}

		addExpenseToCategory(categories, ex, categoryName)
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
			category.LastTransaction = category.Expenses[0].Date()
			// Calculate average amount
			total := int64(0)
			for _, exp := range category.Expenses {
				total += exp.Amount()

				if exp.Date().After(category.LastTransaction) {
					category.LastTransaction = exp.Date()
				}
			}
			category.AvgAmount = total / int64(len(category.Expenses))
		}

		// Calculate budget information for expense categories only
		if category.Amount < 0 {
			budget, hasBudget := categoryBudgets[key]
			if hasBudget && budget > 0 {
				category.Budget = calculateBudgetInfo(budget, category.Amount)
			} else {
				category.Budget = BudgetInfo{
					Amount: 0,
					Status: BudgetStatusNoBudget,
				}
			}
		} else {
			// Income categories don't have budgets
			category.Budget = BudgetInfo{
				Amount: 0,
				Status: BudgetStatusNoBudget,
			}
		}

		categories[key] = category
	}

	return categories, duplicates, income, spending, nil
}

func addExpenseToCategory(categories map[string]Category, ex pkgStorage.Expense, categoryString string) {
	categoryName := expeseCategoryName(ex, categoryString)

	c, ok := categories[categoryName]
	if ok {
		c.Amount += ex.Amount()
		c.Expenses = append(c.Expenses, ex)
		categories[categoryName] = c
	} else {
		amount := ex.Amount()
		cat := Category{
			Amount: amount,
			Name:   categoryName,
			Expenses: []pkgStorage.Expense{
				ex,
			},
		}
		categories[categoryName] = cat
	}
}

func expeseCategoryName(ex pkgStorage.Expense, categoryName string) string {
	if ex.Type() == pkgStorage.IncomeType {
		return "income"
	}

	if categoryName == "" {
		return "uncategorized charge"
	}
	return categoryName
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

func calculateBudgetInfo(budgetAmount int64, spentAmount int64) BudgetInfo {
	// spentAmount is negative for expenses, convert to positive
	spent := spentAmount * -1

	remaining := budgetAmount - spent

	var percentageUsed float64
	if budgetAmount > 0 {
		percentageUsed = (float64(spent) / float64(budgetAmount)) * 100 //nolint:mnd // the value is obvious
	} else {
		percentageUsed = 0
	}

	var status BudgetStatus
	switch {
	case percentageUsed < budgetUnder:
		status = BudgetStatusUnder
	case percentageUsed <= budgetFull:
		status = BudgetStatusNear
	default:
		status = BudgetStatusOver
	}

	return BudgetInfo{
		Amount:         budgetAmount,
		Spent:          spent,
		Remaining:      remaining,
		PercentageUsed: percentageUsed,
		Status:         status,
	}
}
