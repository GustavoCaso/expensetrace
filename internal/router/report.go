package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/domain"
)

type reportHandler struct {
	*router
}

func newReportsHandlder(router *router) *reportHandler {
	return &reportHandler{
		router,
	}
}

func (rh *reportHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		rh.reportService.Generate(r.Context(), userIDFromContext(r.Context()))
		rh.reportsHandler(w, r)
	})
}

func (rh *reportHandler) reportsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := userIDFromContext(ctx)
	base := viewBaseFromContext(ctx)
	data := domain.HomeViewData{
		ViewBase: base,
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
		rep := rh.reportService.ForMonth(userID, selectedMonth, selectedYear)
		openCategory := query.Get("open_category")
		rh.renderHTML(w, http.StatusOK, domain.ReportCardData{
			Report:       rep,
			OpenCategory: openCategory,
			OpenMonth:    selectedMonth,
			OpenYear:     selectedYear,
		}, "reports/card")
		return
	}

	// Full page — chart + optional pre-rendered card via open_month/open_year/open_category
	chartData := rh.reportService.ChartData(userID)
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

	data.ReportCard = domain.ReportCardData{
		Report:       rh.reportService.ForMonth(userID, selectedMonth, selectedYear),
		OpenCategory: openCategory,
		OpenMonth:    selectedMonth,
		OpenYear:     selectedYear,
	}

	rh.renderHTML(w, http.StatusOK, data, "base", "pages/reports/index.html")
}
