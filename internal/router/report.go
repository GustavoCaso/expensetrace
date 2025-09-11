package router

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/util"
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

type reportHandler struct {
	*router
}

func (rh *reportHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		rh.reportsOnce.Do(func() {
			rh.generateReports()
		})
		rh.reportsHandler(w, r)
	})
}

func (rh *reportHandler) generateChartData() []chartDataPoint {
	chartData := make([]chartDataPoint, 0, len(rh.sortedReportKeys))

	for _, key := range rh.sortedReportKeys {
		parts := strings.Split(key, "-")

		r := rh.reports[key]
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

func (rh *reportHandler) reportsHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	var month int
	var year int
	var err error
	useReportTemplate := false

	monthQuery := r.URL.Query().Get("month")
	if monthQuery != "" {
		if month, err = strconv.Atoi(monthQuery); err != nil {
			rh.logger.Warn(fmt.Sprintf("error strconv.Atoi with value %s. %s", monthQuery, err.Error()))
			month = int(now.Month() - 1)
		}
		useReportTemplate = true
	} else {
		month = int(now.Month() - 1)
	}
	yearQuery := r.URL.Query().Get("year")
	if yearQuery != "" {
		if year, err = strconv.Atoi(yearQuery); err != nil {
			rh.logger.Warn(fmt.Sprintf("error strconv.Atoi with value %s. %s", yearQuery, err.Error()))
			year = now.Year()
		}
		useReportTemplate = true
	} else {
		year = now.Year()
	}

	var chartData []chartDataPoint
	if !useReportTemplate {
		chartData = rh.generateChartData()
	}

	data := homeViewData{}
	if err != nil {
		data.Error = err.Error()
	} else {
		reportKey := fmt.Sprintf("%d-%d", year, month)
		report, ok := rh.reports[reportKey]

		if !ok {
			data.Error = fmt.Sprintf("no report available. %s", reportKey)
		} else {
			data.Report = report
			data.ChartData = chartData
		}
	}

	if useReportTemplate {
		rh.templates.Render(w, "partials/reports/card.html", data.Report)
	} else {
		rh.templates.Render(w, "pages/reports/index.html", data)
	}
}

func (rh *reportHandler) generateReports() {
	now := time.Now()
	month := now.Month()
	year := now.Year()
	skipYear := false
	ex, err := rh.storage.GetFirstExpense()
	if err != nil {
		rh.logger.Warn("Failed to generate reports", "error", err)
		return
	}

	lastMonth := ex.Date().Month()
	lastYear := ex.Date().Year()

	reports := map[string]report.Report{}

	for year >= lastYear {
		if month == time.January {
			skipYear = true
		}

		firstDay, lastDay := util.GetMonthDates(int(month), year)

		expenses, expenseErr := rh.storage.GetExpensesFromDateRange(firstDay, lastDay)

		if expenseErr != nil {
			rh.logger.Warn("Failed to generate reports", "error", expenseErr)
			return
		}

		result, reportErr := report.Generate(firstDay, lastDay, rh.storage, expenses, "monthly")

		if reportErr != nil {
			rh.logger.Warn("Failed to generate reports", "error", reportErr)
			return
		}

		reports[fmt.Sprintf("%d-%d", year, month)] = result

		if skipYear {
			year--
			month = time.December
			skipYear = false
			continue
		}

		if year == lastYear && month == lastMonth {
			break
		}

		month--
	}

	reportKeys := slices.Collect(maps.Keys(reports))

	sort.SliceStable(reportKeys, func(i, j int) bool {
		s1 := strings.Split(reportKeys[i], "-")
		s2 := strings.Split(reportKeys[j], "-")
		year1, _ := strconv.Atoi(s1[0])
		month1, _ := strconv.Atoi(s1[1])

		year2, _ := strconv.Atoi(s2[0])
		month2, _ := strconv.Atoi(s2[1])

		if year1 == year2 {
			return time.Month(month1) > time.Month(month2)
		}

		return year1 > year2
	})

	rh.reports = reports
	rh.sortedReportKeys = reportKeys
}
