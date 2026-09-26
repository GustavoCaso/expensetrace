package domain

import (
	"net/url"
	"strconv"
	"time"
)

type BudgetStatus string

const (
	BudgetStatusUnder    BudgetStatus = "under"
	BudgetStatusNear     BudgetStatus = "near"
	BudgetStatusOver     BudgetStatus = "over"
	BudgetStatusNoBudget BudgetStatus = "no_budget"
)

const (
	BudgetUnder = 80
	BudgetFull  = 100
)

type BudgetInfo struct {
	Amount         int64        `json:"amount"`          // Budget amount in cents (0 if no budget)
	Spent          int64        `json:"spent"`           // Amount spent (positive)
	Remaining      int64        `json:"remaining"`       // Remaining budget (can be negative)
	PercentageUsed float64      `json:"percentage_used"` // Percentage of budget used
	Status         BudgetStatus `json:"status"`          // Color coding status
}

type CategoryReport struct {
	Name              string     `json:"name"`
	Amount            int64      `json:"amount"`
	Expenses          []Expense  `json:"expenses"`
	PercentageOfTotal float64    `json:"percentage_of_total"`
	LastTransaction   time.Time  `json:"last_transaction"`
	AvgAmount         int64      `json:"average_amount"`
	Budget            BudgetInfo `json:"budget"`
}

type Report struct {
	Title                 string           `json:"title"`
	Spending              int64            `json:"spending"`
	Income                int64            `json:"income"`
	Savings               int64            `json:"savings"`
	StartDate             time.Time        `json:"start_date"`
	EndDate               time.Time        `json:"end_date"`
	SavingsPercentage     float32          `json:"savings_percentage"`
	EarningsPerDay        int64            `json:"earnings_per_day"`
	AverageSpendingPerDay int64            `json:"average_spending_per_day"`
	ExpenseCategories     []CategoryReport `json:"expense_categories"`
	IncomeCategories      []CategoryReport `json:"income_categories"`
}

// ChartDataPoint represents a single point in the spending/income chart.
type ChartDataPoint struct {
	Month             string  `json:"Month"`
	URL               string  `json:"URL"`
	Income            int64   `json:"Income"`
	Spending          int64   `json:"Spending"`
	Savings           int64   `json:"Savings"`
	SavingsPercentage float32 `json:"SavingsPercentage"`
}

// HomeViewData is the view data for the reports home page.
type HomeViewData struct {
	ViewBase
	ChartData  []ChartDataPoint
	ReportCard ReportCardData
}

// ReportCardData is the view data for a single report card, optionally
// scoped to an open category.
type ReportCardData struct {
	Report
	OpenCategory string
	OpenMonth    int
	OpenYear     int
}

// CategorySection groups the categories of one kind (expenses or income)
// for the report card breakdown.
type CategorySection struct {
	Title      string
	Kind       string // "expense" or "income"; doubles as the CSS color class
	Total      int64
	Categories []CategoryReport
}

// CategorySections returns the non-empty category sections, expenses first.
func (d ReportCardData) CategorySections() []CategorySection {
	sections := []CategorySection{}
	if len(d.ExpenseCategories) > 0 {
		sections = append(sections, CategorySection{
			Title:      "Expenses",
			Kind:       "expense",
			Total:      d.Spending,
			Categories: d.ExpenseCategories,
		})
	}
	if len(d.IncomeCategories) > 0 {
		sections = append(sections, CategorySection{
			Title:      "Income",
			Kind:       "income",
			Total:      d.Income,
			Categories: d.IncomeCategories,
		})
	}
	return sections
}

// CategoryRedirect returns the reports URL that reopens this report card with
// the given category expanded. Used as redirect_to when editing an expense.
func (d ReportCardData) CategoryRedirect(category string) string {
	query := url.Values{}
	query.Set("open_month", strconv.Itoa(d.OpenMonth))
	query.Set("open_year", strconv.Itoa(d.OpenYear))
	query.Set("open_category", category)
	return "/?" + query.Encode()
}
