package router

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/report"
)

type chartDataPoint struct {
	Month             string  `json:"Month"`
	URL               string  `json:"URL"`
	Income            int64   `json:"Income"`
	Spending          int64   `json:"Spending"`
	Savings           int64   `json:"Savings"`
	SavingsPercentage float32 `json:"SavingsPercentage"`
}

type homeViewData struct {
	viewBase
	Report    report.Report
	ChartData []chartDataPoint
}

func (router *router) generateChartData() []chartDataPoint {
	chartData := make([]chartDataPoint, 0, len(router.sortedReportKeys))

	for _, key := range router.sortedReportKeys {
		parts := strings.Split(key, "-")

		r := router.reports[key]
		chartData = append(chartData, chartDataPoint{
			Month:             r.Title,
			Income:            r.Income,
			URL:               fmt.Sprintf("/?month=%s&year=%s", parts[1], parts[0]),
			Spending:          r.Spending,
			Savings:           r.Savings,
			SavingsPercentage: r.SavingsPercentage,
		})
	}

	// Reverse the order to have oldest months first (better for chart visualization)
	for i, j := 0, len(chartData)-1; i < j; i, j = i+1, j-1 {
		chartData[i], chartData[j] = chartData[j], chartData[i]
	}

	return chartData
}

func (router *router) homeHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	var month int
	var year int
	var err error
	useReportTemplate := false

	monthQuery := r.URL.Query().Get("month")
	if monthQuery != "" {
		if month, err = strconv.Atoi(monthQuery); err != nil {
			router.logger.Warn(fmt.Sprintf("error strconv.Atoi with value %s. %s", monthQuery, err.Error()))
			month = int(now.Month() - 1)
		}
		useReportTemplate = true
	} else {
		month = int(now.Month() - 1)
	}
	yearQuery := r.URL.Query().Get("year")
	if yearQuery != "" {
		if year, err = strconv.Atoi(yearQuery); err != nil {
			router.logger.Warn(fmt.Sprintf("error strconv.Atoi with value %s. %s", yearQuery, err.Error()))
			year = now.Year()
		}
		useReportTemplate = true
	} else {
		year = now.Year()
	}

	var chartData []chartDataPoint
	if !useReportTemplate {
		chartData = router.generateChartData()
	}

	data := homeViewData{}
	if err != nil {
		data.Error = err.Error()
	} else {
		reportKey := fmt.Sprintf("%d-%d", year, month)
		report, ok := router.reports[reportKey]

		if !ok {
			data.Error = fmt.Sprintf("no report available. %s", reportKey)
		} else {
			data.Report = report
			data.ChartData = chartData
		}
	}

	if useReportTemplate {
		router.templates.Render(w, "partials/reports/card.html", data.Report)
	} else {
		router.templates.Render(w, "pages/reports/index.html", data)
	}
}
