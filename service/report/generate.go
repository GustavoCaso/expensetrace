package report

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/storage"
)

const (
	percentageOfTotal = 100
)

func generate(
	ctx context.Context,
	userID int64,
	startDate, endDate time.Time,
	storage storage.Storage,
	expenses []domain.Expense,
	reportType string,
) (domain.Report, error) {
	var report domain.Report

	expenseCategories, incomeCategories, totalIncome, totalSpending, err := splitByExpenseType(
		ctx,
		userID,
		storage,
		expenses,
	)

	if err != nil {
		return report, err
	}

	report.Income = totalIncome
	report.Spending = totalSpending
	report.StartDate = startDate
	report.EndDate = endDate
	savings := totalIncome - (totalSpending)*-1
	report.Savings = savings
	savingsPercentage := (float32(savings) / float32(totalIncome)) * percentageOfTotal
	if math.IsNaN(float64(savingsPercentage)) || math.IsInf(float64(savingsPercentage), 0) {
		report.SavingsPercentage = 0
	} else {
		report.SavingsPercentage = savingsPercentage
	}
	numberOfDaysPerMonth := calendarDays(startDate, endDate)
	report.AverageSpendingPerDay = (totalSpending) * -1 / int64(numberOfDaysPerMonth)
	report.EarningsPerDay = totalIncome / int64(numberOfDaysPerMonth)

	report.ExpenseCategories = expenseCategories
	report.IncomeCategories = incomeCategories

	if reportType == "monthly" {
		report.Title = fmt.Sprintf("%s %d", startDate.Month().String(), startDate.Year())
	} else {
		report.Title = strconv.Itoa(startDate.Year())
	}

	return report, nil
}

func splitByExpenseType(
	ctx context.Context,
	userID int64,
	storage storage.Storage,
	expenses []domain.Expense,
) ([]domain.CategoryReport, []domain.CategoryReport, int64, int64, error) {
	var incomeTotal int64
	var spendingTotal int64
	// Track category budgets by category name
	categoryBudgets := make(map[string]int64)
	expenseCategories := []domain.CategoryReport{}
	incomeCategories := []domain.CategoryReport{}
	// internal map to keep track of expense categories
	expenseCategoryMap := map[string]domain.CategoryReport{}

	for _, ex := range expenses {
		categoryName := ""
		if ex.CategoryID() != nil {
			category, categoryError := storage.GetCategory(ctx, userID, *ex.CategoryID())

			if categoryError != nil {
				return expenseCategories, incomeCategories, incomeTotal, spendingTotal, categoryError
			}

			if category.Name() == domain.ExcludeCategory {
				continue
			}

			categoryName = category.Name()
			// Store budget for this category
			categoryBudgets[categoryName] = category.MonthlyBudget()
		}

		switch ex.Type() {
		case domain.ChargeType:
			spendingTotal += ex.Amount()
		case domain.IncomeType:
			incomeTotal += ex.Amount()
		}
		addExpenseToCategory(expenseCategoryMap, ex, categoryName)
	}

	for key, category := range expenseCategoryMap {
		// Calculate percentage of total
		if category.Amount < 0 && spendingTotal < 0 {
			// For expense categories
			category.PercentageOfTotal = float64(category.Amount*-percentageOfTotal) / float64(spendingTotal*-1)
		} else if category.Amount > 0 && incomeTotal > 0 {
			// For income categories
			category.PercentageOfTotal = float64(category.Amount*percentageOfTotal) / float64(incomeTotal)
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
				category.Budget = domain.BudgetInfo{
					Amount: 0,
					Status: domain.BudgetStatusNoBudget,
				}
			}
		} else {
			// Income categories don't have budgets
			category.Budget = domain.BudgetInfo{
				Amount: 0,
				Status: domain.BudgetStatusNoBudget,
			}
		}

		if category.Amount < 0 {
			expenseCategories = append(expenseCategories, category)
		} else {
			incomeCategories = append(incomeCategories, category)
		}
	}

	return expenseCategories, incomeCategories, incomeTotal, spendingTotal, nil
}

func addExpenseToCategory(categories map[string]domain.CategoryReport, ex domain.Expense, categoryString string) {
	categoryName := expeseCategoryName(ex, categoryString)

	c, ok := categories[categoryName]
	if ok {
		c.Amount += ex.Amount()
		c.Expenses = append(c.Expenses, ex)

		categories[categoryName] = c
	} else {
		amount := ex.Amount()
		cat := domain.CategoryReport{
			Amount: amount,
			Name:   categoryName,
			Expenses: []domain.Expense{
				ex,
			},
		}
		categories[categoryName] = cat
	}
}

func expeseCategoryName(ex domain.Expense, categoryName string) string {
	if ex.Type() == domain.IncomeType {
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

func calculateBudgetInfo(budgetAmount int64, spentAmount int64) domain.BudgetInfo {
	// spentAmount is negative for expenses, convert to positive
	spent := spentAmount * -1

	remaining := budgetAmount - spent

	var percentageUsed float64
	if budgetAmount > 0 {
		percentageUsed = (float64(spent) / float64(budgetAmount)) * 100 //nolint:mnd // the value is obvious
	} else {
		percentageUsed = 0
	}

	var status domain.BudgetStatus
	switch {
	case percentageUsed < domain.BudgetUnder:
		status = domain.BudgetStatusUnder
	case percentageUsed <= domain.BudgetFull:
		status = domain.BudgetStatusNear
	default:
		status = domain.BudgetStatusOver
	}

	return domain.BudgetInfo{
		Amount:         budgetAmount,
		Spent:          spent,
		Remaining:      remaining,
		PercentageUsed: percentageUsed,
		Status:         status,
	}
}
