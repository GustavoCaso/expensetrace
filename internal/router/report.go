package router

import (
	"context"
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
	ChartData  []chartDataPoint
	ReportCard reportCardData
}

type reportCardData struct {
	report.Report
	OpenCategory string
	OpenMonth    int
	OpenYear     int
}

type reportHandler struct {
	*router
	reportsPerUser map[int64]map[string]report.Report
}

func newReportsHandlder(router *router) *reportHandler {
	reportsPerUser := map[int64]map[string]report.Report{}

	return &reportHandler{
		router,
		reportsPerUser,
	}
}

func (rh *reportHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		rh.generateReports(r.Context())
		rh.reportsHandler(w, r)
	})
}

func (rh *reportHandler) generateChartData(userID int64) []chartDataPoint {
	reports := rh.reportsPerUser[userID]
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

	chartData := make([]chartDataPoint, 0, len(reports))

	for _, key := range reportKeys {
		parts := strings.Split(key, "-")
		report := reports[key]

		chartData = append(chartData, chartDataPoint{
			Month:             report.Title,
			Income:            report.Income,
			URL:               fmt.Sprintf("/?month=%s&year=%s", parts[1], parts[0]),
			Spending:          report.Spending,
			Savings:           report.Savings,
			SavingsPercentage: report.SavingsPercentage,
		})
	}

	// Reverse the order to have oldest months first (better for chart visualization)
	for i, j := 0, len(chartData)-1; i < j; i, j = i+1, j-1 {
		chartData[i], chartData[j] = chartData[j], chartData[i]
	}

	return chartData
}

func (rh *reportHandler) reportsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)
	base := newViewBase(ctx, rh.storage, rh.logger, pageReports)
	data := homeViewData{
		viewBase: base,
	}

	now := time.Now()
	selectedYear := now.Year()
	selectedMonth := int(now.Month())
	query := r.URL.Query()

	// ?month=X&year=Y — HTMX partial swap only
	if monthQuery := query.Get("month"); monthQuery != "" {
		if m, err := strconv.Atoi(monthQuery); err == nil {
			selectedMonth = m
		}

		if yearQuery := query.Get("year"); yearQuery != "" {
			if y, err := strconv.Atoi(yearQuery); err == nil {
				selectedYear = y
			}
		}
		reportKey := fmt.Sprintf("%d-%d", selectedYear, selectedMonth)
		rep := rh.reportsPerUser[userID][reportKey]
		openCategory := query.Get("open_category")
		rh.templates.Render(w, "partials/reports/card.html", reportCardData{
			Report:       rep,
			OpenCategory: openCategory,
			OpenMonth:    selectedMonth,
			OpenYear:     selectedYear,
		})
		return
	}

	// Full page — chart + optional pre-rendered card via open_month/open_year/open_category
	chartData := rh.generateChartData(userID)
	data.ChartData = chartData

	openCategory := query.Get("open_category")

	if openMonthQuery := query.Get("open_month"); openMonthQuery != "" {
		if m, err := strconv.Atoi(openMonthQuery); err == nil {
			selectedMonth = m
		}
	}
	if openYearQuery := query.Get("open_year"); openYearQuery != "" {
		if y, err := strconv.Atoi(openYearQuery); err == nil {
			selectedYear = y
		}
	}

	reportKey := fmt.Sprintf("%d-%d", selectedYear, selectedMonth)
	data.ReportCard = reportCardData{
		Report:       rh.reportsPerUser[userID][reportKey],
		OpenCategory: openCategory,
		OpenMonth:    selectedMonth,
		OpenYear:     selectedYear,
	}

	rh.templates.Render(w, "pages/reports/index.html", data)
}

func (rh *reportHandler) generateReports(ctx context.Context) {
	userID := userIDFromContext(ctx)
	now := time.Now()
	month := now.Month()
	year := now.Year()
	skipYear := false
	ex, err := rh.storage.GetFirstExpense(ctx, userID)
	if err != nil {
		rh.logger.Warn("Failed to generate reports", "error", err, "userID", userID)
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

		expenses, expenseErr := rh.storage.GetExpensesFromDateRange(ctx, userID, firstDay, lastDay)

		if expenseErr != nil {
			rh.logger.Warn("Failed to generate reports", "error", expenseErr, "userID", userID)
			return
		}

		result, reportErr := report.Generate(ctx, userID, firstDay, lastDay, rh.storage, expenses, "monthly")

		if reportErr != nil {
			rh.logger.Warn("Failed to generate reports", "error", reportErr, "userID", userID)
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

	rh.reportsPerUser[userID] = reports
}
