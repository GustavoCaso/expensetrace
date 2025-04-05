package router

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/report"
)

type link struct {
	Name     string
	URL      string
	Income   int64
	Spending int64
	Savings  int64
}

type homeData struct {
	Report report.Report
	Links  []link
	Error  error
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

	var links []link
	if !useReportTemplate {
		links = router.generateLinks()
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
				Error: fmt.Errorf("No report available. %s", reportKey),
			}
		} else {
			data = homeData{
				Report: report,
				Links:  links,
				Error:  nil,
			}
		}
	}

	if useReportTemplate {
		router.templates.Render(w, "partials/reports/card.html", data.Report)
	} else {
		router.templates.Render(w, "pages/reports/index.html", data)
	}
}

func (router *router) generateLinks() []link {
	links := make([]link, len(router.sortedReportKeys))

	for i, reportKey := range router.sortedReportKeys {
		s := strings.Split(reportKey, "-")
		month, _ := strconv.Atoi(s[1])

		r := router.reports[reportKey]
		links[i] = link{
			Name:     fmt.Sprintf("%s %s", time.Month(month), s[0]),
			URL:      fmt.Sprintf("/?month=%s&year=%s", s[1], s[0]),
			Income:   r.Income,
			Spending: r.Spending,
			Savings:  r.Savings,
		}
	}

	return links
}
