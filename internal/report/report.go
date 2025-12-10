package report

import (
	"context"
	"fmt"
	"math"
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
	Amount         int64        `json:"amount"`          // Budget amount in cents (0 if no budget)
	Spent          int64        `json:"spent"`           // Amount spent (positive)
	Remaining      int64        `json:"remaining"`       // Remaining budget (can be negative)
	PercentageUsed float64      `json:"percentage_used"` // Percentage of budget used
	Status         BudgetStatus `json:"status"`          // Color coding status
}

type Category struct {
	Name              string               `json:"name"`
	Amount            int64                `json:"amount"`
	Expenses          []pkgStorage.Expense `json:"expenses"`
	PercentageOfTotal float64              `json:"percentage_of_total"`
	LastTransaction   time.Time            `json:"last_transaction"`
	AvgAmount         int64                `json:"average_amount"`
	Budget            BudgetInfo           `json:"budget"`
}

type Report struct {
	Title                 string     `json:"title"`
	Spending              int64      `json:"spending"`
	Income                int64      `json:"income"`
	Savings               int64      `json:"savings"`
	StartDate             time.Time  `json:"start_date"`
	EndDate               time.Time  `json:"end_date"`
	SavingsPercentage     float32    `json:"savings_percentage"`
	EarningsPerDay        int64      `json:"earnings_per_day"`
	AverageSpendingPerDay int64      `json:"average_spending_per_day"`
	ExpenseCategories     []Category `json:"expense_categories"`
	IncomeCategories      []Category `json:"income_categories"`
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
	storage pkgStorage.Storage,
	expenses []pkgStorage.Expense,
) ([]Category, []Category, int64, int64, error) {
	var incomeTotal int64
	var spendingTotal int64
	// Track category budgets by category name
	categoryBudgets := make(map[string]int64)
	expenseCategories := []Category{}
	incomeCategories := []Category{}
	// internal map to keep track of expense categories
	expenseCategoryMap := map[string]Category{}

	for _, ex := range expenses {
		categoryName := ""
		if ex.CategoryID() != nil {
			category, categoryError := storage.GetCategory(ctx, userID, *ex.CategoryID())

			if categoryError != nil {
				return expenseCategories, incomeCategories, incomeTotal, spendingTotal, categoryError
			}

			if category.Name() == pkgStorage.ExcludeCategory {
				continue
			}

			categoryName = category.Name()
			// Store budget for this category
			categoryBudgets[categoryName] = category.MonthlyBudget()
		}

		switch ex.Type() {
		case pkgStorage.ChargeType:
			spendingTotal += ex.Amount()
		case pkgStorage.IncomeType:
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

		if category.Amount < 0 {
			expenseCategories = append(expenseCategories, category)
		} else {
			incomeCategories = append(incomeCategories, category)
		}
	}

	return expenseCategories, incomeCategories, incomeTotal, spendingTotal, nil
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
