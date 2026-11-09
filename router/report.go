package router

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
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

	// Full page — period selector, chart for the selected range, then either a
	// month card (open_month/open_year, used by redirects after editing an
	// expense) or the range card (from/to, defaulting to the current year to
	// date).
	openCategory := query.Get("open_category")

	openMonth := query.Get("open_month") != ""
	if openMonth {
		if m, err := strconv.Atoi(query.Get("open_month")); err == nil {
			selectedMonth = m
		}
		if y, err := strconv.Atoi(query.Get("open_year")); err == nil {
			selectedYear = y
		}
	}

	// The chart shows the selected range; a month card shows its whole year.
	from, to := defaultRange(now)
	if openMonth {
		from, to = yearRange(selectedYear, now)
	}
	if f, ok := parseMonthParam(query.Get("from")); ok {
		from = f
	}
	if t, ok := parseMonthParam(query.Get("to")); ok {
		to = t
	}
	if to.Before(from) {
		from, to = to, from
	}

	data.Period = buildPeriodSelector(rh.reportService.Years(userID), from, to, now)

	selection, err := rh.reportService.Range(ctx, userID, from, to)
	if err != nil {
		rh.logger.Error("Failed to generate range report", "error", err, "userID", userID)
		data.Error = "Failed to generate report"
		rh.renderHTML(w, http.StatusInternalServerError, data, "base", "pages/reports/index.html")
		return
	}
	data.ChartData = selection.ChartData()

	if openMonth {
		data.ReportCard = domain.ReportCardData{
			Report:       rh.reportService.ForMonth(userID, selectedMonth, selectedYear),
			OpenCategory: openCategory,
			OpenMonth:    selectedMonth,
			OpenYear:     selectedYear,
		}

		rh.renderHTML(w, http.StatusOK, data, "base", "pages/reports/index.html")
		return
	}

	data.Selection = &domain.SelectionCardData{
		SelectionReport: selection,
		OpenCategory:    openCategory,
		From:            from.Format(monthParamLayout),
		To:              to.Format(monthParamLayout),
	}

	rh.renderHTML(w, http.StatusOK, data, "base", "pages/reports/index.html")
}

// monthParamLayout is the "YYYY-MM" format used by <input type="month">.
const monthParamLayout = "2006-01"

func parseMonthParam(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(monthParamLayout, value, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// defaultRange is January of the current year through the current month.
func defaultRange(now time.Time) (time.Time, time.Time) {
	return yearRange(now.Year(), now)
}

// yearRange spans January to December of year, or January to the current
// month when year is the current year. It matches the year tabs.
func yearRange(year int, now time.Time) (time.Time, time.Time) {
	lastMonth := time.December
	if year == now.Year() {
		lastMonth = now.Month()
	}
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(year, lastMonth, 1, 0, 0, 0, 0, time.Local)
	return from, to
}

// buildPeriodSelector builds one tab per year. A past year spans January to
// December; the current year spans January to the current month. A tab is
// active when the selected range matches its span exactly.
func buildPeriodSelector(years []int, from, to, now time.Time) domain.PeriodSelector {
	selector := domain.PeriodSelector{
		From: from.Format(monthParamLayout),
		To:   to.Format(monthParamLayout),
		Max:  now.Format(monthParamLayout),
	}

	for _, year := range years {
		start, end := yearRange(year, now)
		tabFrom := start.Format(monthParamLayout)
		tabTo := end.Format(monthParamLayout)

		selector.Years = append(selector.Years, domain.YearTab{
			Year:   year,
			URL:    "/?" + url.Values{"from": {tabFrom}, "to": {tabTo}}.Encode(),
			Active: selector.From == tabFrom && selector.To == tabTo,
		})
	}

	return selector
}
