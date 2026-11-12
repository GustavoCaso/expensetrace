package report

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/storage"
)

const (
	percentageOfTotal = 100
)

// generate builds a report for the expenses between startDate and endDate.
// The range may span several months: monthly budgets are scaled by the
// number of calendar months covered, and the title reflects the range.
func generate(
	ctx context.Context,
	userID int64,
	startDate, endDate time.Time,
	storage storage.Storage,
	expenses []domain.Expense,
) (domain.Report, error) {
	var report domain.Report

	months := monthsInRange(startDate, endDate)

	expenseCategories, incomeCategories, totalIncome, totalSpending, err := splitByExpenseType(
		ctx,
		userID,
		storage,
		expenses,
		months,
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
	// The range is inclusive of both days: January 1–31 is 31 days, not the
	// 30-day difference calendarDays reports.
	daysInRange := int64(calendarDays(startDate, endDate) + 1)
	report.AverageSpendingPerDay = (totalSpending) * -1 / daysInRange
	report.EarningsPerDay = totalIncome / daysInRange

	report.ExpenseCategories = expenseCategories
	report.IncomeCategories = incomeCategories
	report.MonthCount = months
	report.Title = reportTitle(startDate, endDate)

	return report, nil
}

// monthsInRange counts the calendar months touched by [start, end], inclusive.
// It never returns less than 1.
func monthsInRange(start, end time.Time) int {
	months := (end.Year()-start.Year())*monthsInYear + int(end.Month()) - int(start.Month()) + 1
	return max(months, 1)
}

const monthsInYear = 12

// reportTitle names a report after its range: "November 2026" for a single
// month, "2026" for a full calendar year, "Jan – Nov 2026" within one year,
// and "Mar 2025 – Feb 2026" across years.
func reportTitle(start, end time.Time) string {
	sameYear := start.Year() == end.Year()

	switch {
	case sameYear && start.Month() == end.Month():
		return fmt.Sprintf("%s %d", start.Month(), start.Year())
	case sameYear && start.Month() == time.January && end.Month() == time.December:
		return strconv.Itoa(start.Year())
	case sameYear:
		return fmt.Sprintf("%s – %s %d", start.Format("Jan"), end.Format("Jan"), end.Year())
	default:
		return fmt.Sprintf("%s – %s", start.Format("Jan 2006"), end.Format("Jan 2006"))
	}
}

func splitByExpenseType(
	ctx context.Context,
	userID int64,
	storage storage.Storage,
	expenses []domain.Expense,
	months int,
) ([]domain.CategoryReport, []domain.CategoryReport, int64, int64, error) {
	var incomeTotal int64
	var spendingTotal int64
	// Track category budgets by category name
	categoryBudgets := make(map[string]int64)
	expenseCategories := []domain.CategoryReport{}
	incomeCategories := []domain.CategoryReport{}
	// internal map to keep track of expense categories
	expenseCategoryMap := map[string]domain.CategoryReport{}

	// Load categories once; multi-month ranges can hold thousands of expenses.
	allCategories, err := storage.GetCategories(ctx, userID)
	if err != nil {
		return expenseCategories, incomeCategories, incomeTotal, spendingTotal, err
	}
	categoriesByID := make(map[int64]domain.Category, len(allCategories))
	for _, c := range allCategories {
		categoriesByID[c.ID()] = c
	}

	for _, ex := range expenses {
		categoryName := ""
		if ex.CategoryID() != nil {
			category, found := categoriesByID[*ex.CategoryID()]
			if !found {
				return expenseCategories, incomeCategories, incomeTotal, spendingTotal,
					fmt.Errorf("category %d not found", *ex.CategoryID())
			}

			if category.Name() == domain.ExcludeCategory {
				continue
			}

			// A category only groups expenses of a type it accepts. Anything
			// else (legacy data) is reported as uncategorized, so income never
			// nets against spending in a charge category.
			if category.Type().Accepts(ex.Type()) {
				categoryName = category.Name()
				// Store the budget for the whole range, not just one month.
				categoryBudgets[categoryName] = category.MonthlyBudget() * int64(months)
			}
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
			if incomeTotal > 0 {
				category.PercentageOfIncome = float64(category.Amount*-percentageOfTotal) / float64(incomeTotal)
			}
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
				category.Budget = CalculateBudgetInfo(budget, category.Amount)
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

	sortByAmount(expenseCategories)
	sortByAmount(incomeCategories)

	return expenseCategories, incomeCategories, incomeTotal, spendingTotal, nil
}

// sortByAmount orders categories by absolute amount, largest first,
// falling back to name so the order is stable across renders.
func sortByAmount(categories []domain.CategoryReport) {
	slices.SortFunc(categories, func(a, b domain.CategoryReport) int {
		if c := cmp.Compare(absAmount(b.Amount), absAmount(a.Amount)); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})
}

func absAmount(amount int64) int64 {
	if amount < 0 {
		return -amount
	}
	return amount
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
	if categoryName != "" {
		return categoryName
	}

	if ex.Type() == domain.IncomeType {
		return "uncategorized income"
	}
	return "uncategorized charge"
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

// CalculateBudgetInfo returns the budget status for spentAmount (negative for
// charges) against budgetAmount. Also used by the categories page.
func CalculateBudgetInfo(budgetAmount int64, spentAmount int64) domain.BudgetInfo {
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
