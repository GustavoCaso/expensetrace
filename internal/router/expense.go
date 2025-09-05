package router

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
)

type expensesByYear map[int]map[string][]*expenseDB.Expense

const centsMultiplier = 100

var months = []string{
	time.December.String(),
	time.November.String(),
	time.October.String(),
	time.September.String(),
	time.August.String(),
	time.July.String(),
	time.June.String(),
	time.May.String(),
	time.April.String(),
	time.March.String(),
	time.February.String(),
	time.January.String(),
}

type expesesViewData struct {
	viewBase
	Query        string
	Expenses     expensesByYear
	Years        []int
	Months       []string
	CurrentYear  int
	CurrentMonth string
}

func (router *router) expensesHandler(w http.ResponseWriter) {
	data := expesesViewData{}
	expenses, err := expenseDB.GetExpenses(router.db)
	if err != nil {
		data.Error = err.Error()
		router.templates.Render(w, "pages/expenses/index.html", data)
		return
	}

	groupedExpenses, years := expensesGroupByYearAndMonth(expenses)
	today := time.Now()

	data.Expenses = groupedExpenses
	data.Years = years
	data.Months = months
	data.CurrentYear = today.Year()
	data.CurrentMonth = today.Month().String()

	router.templates.Render(w, "pages/expenses/index.html", data)
}

func expensesGroupByYearAndMonth(expenses []*expenseDB.Expense) (expensesByYear, []int) {
	groupedExpenses := expensesByYear{}
	years := []int{}

	for _, expense := range expenses {
		expenseYear := expense.Date.Year()
		expenseMonth := expense.Date.Month().String()

		year, okYear := groupedExpenses[expenseYear]

		if okYear {
			month, okMonth := year[expenseMonth]
			if okMonth {
				month = append(month, expense)
			} else {
				month = []*expenseDB.Expense{
					expense,
				}
			}

			year[expenseMonth] = month
		} else {
			years = append(years, expenseYear)
			newYear := map[string][]*expenseDB.Expense{}
			newYear[expenseMonth] = []*expenseDB.Expense{
				expense,
			}
			groupedExpenses[expenseYear] = newYear
		}
	}

	sort.SliceStable(years, func(i, j int) bool {
		return years[i] > years[j]
	})

	return groupedExpenses, years
}

type expenseViewData struct {
	viewBase
	Expense    *expenseDB.Expense
	Categories []expenseDB.Category
	FormErrors map[string]string
}

func (router *router) expenseHandler(w http.ResponseWriter, r *http.Request) {
	data := expenseViewData{}
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		data.Error = err.Error()
		router.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	expense, err := expenseDB.GetExpense(router.db, id)
	if err != nil {
		data.Error = err.Error()
		router.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	categories, err := expenseDB.GetCategories(router.db)
	if err != nil {
		router.logger.Error("Failed to get categories", "error", err)
		categories = []expenseDB.Category{}
	}

	data.Expense = expense
	data.Categories = categories

	router.templates.Render(w, "pages/expenses/edit.html", data)
}

func (router *router) updateExpenseHandler(w http.ResponseWriter, r *http.Request) {
	data := expenseViewData{}
	data.FormErrors = make(map[string]string)

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)

	if err != nil {
		data.Error = err.Error()
		router.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	err = r.ParseForm()
	if err != nil {
		router.logger.Error("Failed to parse form", "error", err)
		data.Error = err.Error()
		router.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	source := r.FormValue("source")
	description := r.FormValue("description")
	amountStr := r.FormValue("amount")
	currency := r.FormValue("currency")
	dateStr := r.FormValue("date")
	typeStr := r.FormValue("type")
	categoryIDStr := r.FormValue("category_id")

	if source == "" {
		data.FormErrors["source"] = "Source is required"
	}
	if description == "" {
		data.FormErrors["description"] = "Description is required"
	}
	if currency == "" {
		data.FormErrors["currency"] = "Currency is required"
	}

	var amount int64
	if amountStr == "" {
		data.FormErrors["amount"] = "Amount is required"
	} else {
		amountFloat, parseErr := strconv.ParseFloat(amountStr, 64)
		if parseErr != nil {
			data.FormErrors["amount"] = "Invalid amount format"
		} else {
			amount = int64(amountFloat * centsMultiplier)
		}
	}

	var date time.Time
	if dateStr == "" {
		data.FormErrors["date"] = "Date is required"
	} else {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			data.FormErrors["date"] = "Invalid date format"
		}
	}

	var expenseType expenseDB.ExpenseType
	if typeStr == "" {
		data.FormErrors["type"] = "Type is required"
	} else {
		typeInt, parseErr := strconv.Atoi(typeStr)
		if parseErr != nil {
			data.FormErrors["type"] = "Invalid type"
		} else {
			expenseType = expenseDB.ExpenseType(typeInt)
		}
	}

	var categoryID sql.NullInt64
	if categoryIDStr != "" {
		catID, parseErr := strconv.ParseInt(categoryIDStr, 10, 64)
		if parseErr != nil {
			data.FormErrors["category_id"] = "Invalid category"
		} else {
			categoryID = sql.NullInt64{Int64: catID, Valid: true}
		}
	}

	categories, _ := expenseDB.GetCategories(router.db)
	expense, _ := expenseDB.GetExpense(router.db, id)
	data.Expense = expense
	data.Categories = categories

	if len(data.FormErrors) > 0 {
		router.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	updatedExpense := &expenseDB.Expense{
		ID:          int(id),
		Source:      source,
		Description: description,
		Amount:      amount,
		Date:        date,
		Type:        expenseType,
		Currency:    currency,
		CategoryID:  categoryID,
	}

	updated, err := expenseDB.UpdateExpense(router.db, updatedExpense)
	if err != nil {
		router.logger.Error("Failed to update expense", "error", err, "id", id)
		data.FormErrors["failed to update expense"] = err.Error()
		router.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	if updated != 1 {
		router.logger.Error("Failed to update expense", "id", id)
		data.FormErrors["failed to update expense"] = "No record updated"
		router.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	router.logger.Info("Expense updated successfully", "id", id)

	router.resetCache()
	data.Expense = updatedExpense
	data.Banner = "Expense Updated"
	router.templates.Render(w, "pages/expenses/edit.html", data)
}

func (router *router) deleteExpenseHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		data := expenseViewData{}
		data.Error = fmt.Sprintf("Invalid the ID. %s", err.Error())
		router.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	_, err = expenseDB.DeleteExpense(router.db, id)
	if err != nil {
		router.logger.Error("Failed to delete expense", "error", err, "id", id)

		data := expenseViewData{}
		data.Error = fmt.Sprintf("Error deleting the expense. %s", err.Error())
		router.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	router.logger.Info("Expense deleted successfully", "id", id)

	router.resetCache()

	w.Header().Set("Hx-Redirect", "/expenses")
}

func (router *router) expenseSearchHandler(w http.ResponseWriter, r *http.Request) {
	data := expesesViewData{}
	err := r.ParseForm()

	if err != nil {
		data.Error = err.Error()
		router.templates.Render(w, "pages/expenses/index.html", data)
		return
	}

	query := r.FormValue("q")

	if query == "" {
		data.Error = "You must provide a search criteria"
		router.templates.Render(w, "pages/expenses/index.html", data)
		return
	}

	expenses, err := expenseDB.SearchExpenses(router.db, query)
	if err != nil {
		data.Error = "You must provide a search criteria"
		router.templates.Render(w, "pages/expenses/index.html", data)
	}

	groupedExpenses, years := expensesGroupByYearAndMonth(expenses)
	today := time.Now()

	data.Expenses = groupedExpenses
	data.Years = years
	data.Months = months
	data.CurrentYear = today.Year()
	data.CurrentMonth = today.Month().String()
	data.Query = query

	router.templates.Render(w, "pages/expenses/index.html", data)
}
