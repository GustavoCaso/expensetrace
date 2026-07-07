package domain

import "github.com/GustavoCaso/expensetrace/internal/report"

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
	report.Report
	OpenCategory string
	OpenMonth    int
	OpenYear     int
}
