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

type homeData struct {
	Report    report.Report
	ChartData []chartDataPoint
	Error     error
}

func (router *router) generateChartData() []chartDataPoint {
	chartData := make([]chartDataPoint, 0, len(router.sortedReportKeys))

	for _, key := range router.sortedReportKeys {
		parts := strings.Split(key, "-")
		if len(parts) != 2 {
			continue
		}

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
			fmt.Println("error strconv.Atoi ", err.Error())
			month = int(now.Month() - 1)
		}
		useReportTemplate = true
	} else {
		month = int(now.Month() - 1)
	}
	yearQuery := r.URL.Query().Get("year")
	if yearQuery != "" {
		if year, err = strconv.Atoi(yearQuery); err != nil {
			fmt.Println("error strconv.Atoi ", err.Error())
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

	var data homeData
	if err != nil {
		data = homeData{
			Error: err,
		}
	} else {
		reportKey := fmt.Sprintf("%d-%d", year, month)
		report, ok := router.reports[reportKey]

		if !ok {
			data = homeData{
				Error: fmt.Errorf("no report available. %s", reportKey),
			}
		} else {
			data = homeData{
				Report:    report,
				ChartData: chartData,
				Error:     nil,
			}
		}
	}

	if useReportTemplate {
		router.templates.Render(w, "partials/reports/card.html", data.Report)
	} else {
		router.templates.Render(w, "pages/reports/index.html", data)
	}
}
