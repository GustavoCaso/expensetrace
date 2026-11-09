package domain

import (
	"fmt"
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
	Name              string    `json:"name"`
	Amount            int64     `json:"amount"`
	Expenses          []Expense `json:"expenses"`
	PercentageOfTotal float64   `json:"percentage_of_total"`
	// PercentageOfIncome is an expense category's spending as a share of the
	// period's total income. Zero for income categories or when there was no
	// income. It can exceed 100 when a category outspends all income.
	PercentageOfIncome float64    `json:"percentage_of_income"`
	LastTransaction    time.Time  `json:"last_transaction"`
	AvgAmount          int64      `json:"average_amount"`
	Budget             BudgetInfo `json:"budget"`
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
	// MonthCount is the number of calendar months the report covers (1 for a
	// monthly report). Budgets are already scaled by it.
	MonthCount int `json:"month_count"`
}

// IncomeBarWidth is PercentageOfIncome capped at 100, for drawing a
// progress bar that can't overflow its track.
func (c CategoryReport) IncomeBarWidth() float64 {
	return min(c.PercentageOfIncome, percentageCap)
}

const percentageCap = 100

// PerMonth spreads an amount evenly over the months the report covers.
func (r Report) PerMonth(amount int64) int64 {
	if r.MonthCount <= 1 {
		return amount
	}
	return amount / int64(r.MonthCount)
}

// MonthSummary is the headline figures of one month inside a SelectionReport.
type MonthSummary struct {
	Title             string
	Month             int
	Year              int
	Income            int64
	Spending          int64
	Savings           int64
	SavingsPercentage float32
	// HasData is false for months without any (non-excluded) transactions.
	HasData bool
}

// SelectionReport is a report over a user-selected range of months: the
// combined Report for the whole range plus a summary per month.
type SelectionReport struct {
	Report
	Months []MonthSummary
	// Best and Worst are the months with the highest and lowest savings.
	// Both are nil when fewer than two months have data.
	Best  *MonthSummary
	Worst *MonthSummary
}

// ChartData returns one chart point per month of the selection, oldest
// first. Each point's URL loads that month's report card.
func (s SelectionReport) ChartData() []ChartDataPoint {
	points := make([]ChartDataPoint, 0, len(s.Months))
	for _, m := range s.Months {
		points = append(points, ChartDataPoint{
			Month:             m.Title,
			URL:               fmt.Sprintf("/?month=%d&year=%d", m.Month, m.Year),
			Income:            m.Income,
			Spending:          m.Spending,
			Savings:           m.Savings,
			SavingsPercentage: m.SavingsPercentage,
		})
	}
	return points
}

// IsBest reports whether m is the selection's best month.
func (s SelectionReport) IsBest(m MonthSummary) bool {
	return s.Best != nil && s.Best.Year == m.Year && s.Best.Month == m.Month
}

// IsWorst reports whether m is the selection's worst month.
func (s SelectionReport) IsWorst(m MonthSummary) bool {
	return s.Worst != nil && s.Worst.Year == m.Year && s.Worst.Month == m.Month
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

// HomeViewData is the view data for the reports home page. Exactly one of
// ReportCard (a single month) or Selection (a range of months) is rendered.
type HomeViewData struct {
	ViewBase
	ChartData  []ChartDataPoint
	ReportCard ReportCardData
	Selection  *SelectionCardData
	Period     PeriodSelector
}

// YearTab is one year preset in the reports period selector.
type YearTab struct {
	Year   int
	URL    string
	Active bool
}

// PeriodSelector is the view data for the year tabs and custom range form.
// From, To and Max use the "YYYY-MM" format of <input type="month">.
type PeriodSelector struct {
	Years []YearTab
	From  string
	To    string
	Max   string
}

// SelectionCardData is the view data for a range report card, optionally
// scoped to an open category.
type SelectionCardData struct {
	SelectionReport
	OpenCategory string
	From         string // YYYY-MM
	To           string // YYYY-MM
}

// CategoryRedirect returns the reports URL that reopens this range report
// with the given category expanded. Used as redirect_to when editing an expense.
func (d SelectionCardData) CategoryRedirect(category string) string {
	query := url.Values{}
	query.Set("from", d.From)
	query.Set("to", d.To)
	query.Set("open_category", category)
	return "/?" + query.Encode()
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
func (r Report) CategorySections() []CategorySection {
	sections := []CategorySection{}
	if len(r.ExpenseCategories) > 0 {
		sections = append(sections, CategorySection{
			Title:      "Expenses",
			Kind:       "expense",
			Total:      r.Spending,
			Categories: r.ExpenseCategories,
		})
	}
	if len(r.IncomeCategories) > 0 {
		sections = append(sections, CategorySection{
			Title:      "Income",
			Kind:       "income",
			Total:      r.Income,
			Categories: r.IncomeCategories,
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
