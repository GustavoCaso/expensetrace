package domain

import "testing"

func TestSelectionReportChartData(t *testing.T) {
	sel := SelectionReport{
		Months: []MonthSummary{
			{
				Title:             "December 2025",
				Month:             12,
				Year:              2025,
				Income:            300,
				Spending:          -100,
				Savings:           200,
				SavingsPercentage: 66.6,
			},
			{Title: "January 2026", Month: 1, Year: 2026},
		},
	}

	points := sel.ChartData()

	if len(points) != 2 {
		t.Fatalf("len(points) = %d, want 2", len(points))
	}

	first := points[0]
	if first.Month != "December 2025" || first.URL != "/?month=12&year=2025" {
		t.Errorf("first point = %+v, want December 2025 linking to its month card", first)
	}
	if first.Income != 300 || first.Spending != -100 || first.Savings != 200 || first.SavingsPercentage != 66.6 {
		t.Errorf("first point figures = %+v, want copied from the month summary", first)
	}

	// Months without data still get a point so the chart covers the full range.
	if points[1].Month != "January 2026" || points[1].URL != "/?month=1&year=2026" {
		t.Errorf("second point = %+v, want empty January 2026", points[1])
	}
}
