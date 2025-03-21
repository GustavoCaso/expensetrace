package router

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/util"
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

	firstDay, lastDay := util.GetMonthDates(month, year)
	var links []link
	if !useReportTemplate {
		links, err = generateLinks(router.db, time.Month(month), year)
	}

	var data homeData
	if err != nil {
		data = homeData{
			Error: err,
		}
	} else {
		expenses, err := expenseDB.GetExpensesFromDateRange(router.db, firstDay, lastDay)

		if err != nil {
			data = homeData{
				Error: err,
			}
		} else {
			result := report.Generate(firstDay, lastDay, expenses, "monthly")

			data = homeData{
				Report: result,
				Links:  links,
				Error:  nil,
			}
		}
	}

	if useReportTemplate {
		err = router.templates.Render(w, "partials/reports/report.html", data)
	} else {
		err = router.templates.Render(w, "pages/reports/index.html", data)
	}

	if err != nil {
		log.Print(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func generateLinks(db *sql.DB, month time.Month, year int) ([]link, error) {
	links := []link{}
	skipYear := false
	timeMonth := time.Month(month)
	ex, err := expenseDB.GetFirstExpense(db)
	if err != nil {
		return links, err
	}

	lastMonth := ex.Date.Month()
	lastYear := ex.Date.Year()

	for year >= lastYear {
		if timeMonth == time.January {
			skipYear = true
		}

		firstDay, lastDay := util.GetMonthDates(int(timeMonth), year)

		expenses, err := expenseDB.GetExpensesFromDateRange(db, firstDay, lastDay)

		if err != nil {
			return links, err
		}

		result := report.Generate(firstDay, lastDay, expenses, "monthly")

		links = append(links, link{
			Name:     fmt.Sprintf("%s %d", timeMonth, year),
			URL:      fmt.Sprintf("/?month=%d&year=%d", int(timeMonth), year),
			Income:   result.Income,
			Spending: result.Spending,
			Savings:  result.Savings,
		})

		if skipYear {
			year = year - 1
			timeMonth = time.December
			skipYear = false
			continue
		}

		if year == lastYear && timeMonth == lastMonth {
			break
		}

		timeMonth = timeMonth - 1
	}

	return links, nil
}
