package router

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	pkgStorage "github.com/GustavoCaso/expensetrace/internal/storage"
)

type expenseView struct {
	pkgStorage.Expense
	category pkgStorage.Category
}

func (e *expenseView) Category() pkgStorage.Category {
	return e.category
}

func (e *expenseView) CategoryID() int64 {
	if e.Expense.CategoryID() != nil {
		return *e.Expense.CategoryID()
	}
	return 0
}

type expensesByYear map[int]map[string][]*expenseView

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

type expenseHandler struct {
	*router
}

func (c *expenseHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /expense/{id}", func(w http.ResponseWriter, r *http.Request) {
		c.expenseHandler(w, r)
	})

	mux.HandleFunc("PUT /expense/{id}", func(w http.ResponseWriter, r *http.Request) {
		c.updateExpenseHandler(w, r)
	})

	mux.HandleFunc("DELETE /expense/{id}", func(w http.ResponseWriter, r *http.Request) {
		c.deleteExpenseHandler(w, r)
	})

	mux.HandleFunc("GET /expenses", func(w http.ResponseWriter, _ *http.Request) {
		c.expensesHandler(w, nil)
	})

	mux.HandleFunc("POST /expense/search", func(w http.ResponseWriter, r *http.Request) {
		c.expenseSearchHandler(w, r)
	})
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

func (c *expenseHandler) expensesHandler(w http.ResponseWriter, banner *banner) {
	data := expesesViewData{}
	expenses, err := c.storage.GetAllExpenseTypes()
	if err != nil {
		data.Error = err.Error()
		c.templates.Render(w, "pages/expenses/index.html", data)
		return
	}

	groupedExpenses, years, err := expensesGroupByYearAndMonth(expenses, c.storage)
	if err != nil {
		data.Error = fmt.Sprintf("Error grouping expenses: %s", err.Error())
		c.templates.Render(w, "pages/expenses/index.html", data)
	}

	today := time.Now()

	data.Expenses = groupedExpenses
	data.Years = years
	data.Months = months
	data.CurrentYear = today.Year()
	data.CurrentMonth = today.Month().String()

	if banner != nil {
		data.Banner = *banner
	}

	c.templates.Render(w, "pages/expenses/index.html", data)
}

func expensesGroupByYearAndMonth(
	expenses []pkgStorage.Expense,
	storage pkgStorage.Storage,
) (expensesByYear, []int, error) {
	groupedExpenses := expensesByYear{}
	years := []int{}

	for _, expense := range expenses {
		var category pkgStorage.Category

		if expense.CategoryID() != nil {
			c, categoryErr := storage.GetCategory(*expense.CategoryID())
			if categoryErr != nil {
				if !errors.Is(categoryErr, &pkgStorage.NotFoundError{}) {
					return groupedExpenses, years, categoryErr
				}
			}
			category = c
		}

		expenseYear := expense.Date().Year()
		expenseMonth := expense.Date().Month().String()

		year, okYear := groupedExpenses[expenseYear]

		if okYear {
			month, okMonth := year[expenseMonth]
			if okMonth {
				expenseview := &expenseView{
					Expense:  expense,
					category: category,
				}
				month = append(month, expenseview)
			} else {
				expenseview := &expenseView{
					Expense:  expense,
					category: category,
				}
				month = []*expenseView{expenseview}
			}

			year[expenseMonth] = month
		} else {
			years = append(years, expenseYear)
			newYear := map[string][]*expenseView{}
			expenseview := &expenseView{
				Expense:  expense,
				category: category,
			}
			newYear[expenseMonth] = []*expenseView{expenseview}
			groupedExpenses[expenseYear] = newYear
		}
	}

	sort.SliceStable(years, func(i, j int) bool {
		return years[i] > years[j]
	})

	return groupedExpenses, years, nil
}

type expenseViewData struct {
	viewBase
	Expense    *expenseView
	Categories []pkgStorage.Category
	FormErrors map[string]string
}

func (c *expenseHandler) expenseHandler(w http.ResponseWriter, r *http.Request) {
	data := expenseViewData{}
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		data.Error = err.Error()
		c.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	expense, err := c.storage.GetExpenseByID(id)
	if err != nil {
		data.Error = err.Error()
		c.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	categories, err := c.storage.GetCategories()
	if err != nil {
		c.logger.Error("Failed to get categories", "error", err)
		categories = []pkgStorage.Category{}
	}

	var category pkgStorage.Category
	if expense.CategoryID() != nil {
		cat, categoryErr := c.storage.GetCategory(*expense.CategoryID())
		if categoryErr != nil {
			c.logger.Error("Failed to get category", "error", categoryErr)
		}
		category = cat
	}

	expenseview := &expenseView{
		Expense:  expense,
		category: category,
	}

	data.Expense = expenseview
	data.Categories = categories

	c.templates.Render(w, "pages/expenses/edit.html", data)
}

func (c *expenseHandler) updateExpenseHandler(w http.ResponseWriter, r *http.Request) {
	data := expenseViewData{}
	data.FormErrors = make(map[string]string)

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)

	if err != nil {
		data.Error = err.Error()
		c.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	err = r.ParseForm()
	if err != nil {
		c.logger.Error("Failed to parse form", "error", err)
		data.Error = err.Error()
		c.templates.Render(w, "pages/expenses/edit.html", data)
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

	var expenseType pkgStorage.ExpenseType
	if typeStr == "" {
		data.FormErrors["type"] = "Type is required"
	} else {
		typeInt, parseErr := strconv.Atoi(typeStr)
		if parseErr != nil {
			data.FormErrors["type"] = "Invalid type"
		} else {
			expenseType = pkgStorage.ExpenseType(typeInt)
		}
	}

	var categoryID *int64
	if categoryIDStr != "" {
		catID, parseErr := strconv.ParseInt(categoryIDStr, 10, 64)
		if parseErr != nil {
			data.FormErrors["category_id"] = "Invalid category"
			categoryID = nil
		} else {
			categoryID = &catID
		}
	} else {
		categoryID = nil
	}

	categories, _ := c.storage.GetCategories()
	expense, _ := c.storage.GetExpenseByID(id)
	var category pkgStorage.Category
	if expense.CategoryID() != nil {
		cat, categoryErr := c.storage.GetCategory(*expense.CategoryID())
		if categoryErr != nil {
			c.logger.Error("Failed to get category", "error", categoryErr)
		}
		category = cat
	}

	data.Expense = &expenseView{
		Expense:  expense,
		category: category,
	}
	data.Categories = categories

	if len(data.FormErrors) > 0 {
		c.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	updatedExpense := pkgStorage.NewExpense(id, source, description, currency, amount, date, expenseType, categoryID)

	updated, err := c.storage.UpdateExpense(updatedExpense)
	if err != nil {
		c.logger.Error("Failed to update expense", "error", err, "id", id)
		data.FormErrors["failed to update expense"] = err.Error()
		c.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	if updated != 1 {
		c.logger.Error("Failed to update expense", "id", id)
		data.FormErrors["failed to update expense"] = "No record updated"
		c.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	c.logger.Info("Expense updated successfully", "id", id)

	c.resetCache()
	var updatedCategory pkgStorage.Category
	if updatedExpense.CategoryID() != nil {
		cat, categoryErr := c.storage.GetCategory(*updatedExpense.CategoryID())
		if categoryErr != nil {
			c.logger.Error("Failed to get category", "error", categoryErr)
		}
		updatedCategory = cat
	}

	data.Expense = &expenseView{
		Expense:  updatedExpense,
		category: updatedCategory,
	}

	data.Banner = banner{
		Icon:    "✅",
		Message: "Expense Updated",
	}
	c.templates.Render(w, "pages/expenses/edit.html", data)
}

func (c *expenseHandler) deleteExpenseHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		data := expenseViewData{}
		data.Error = fmt.Sprintf("Invalid the ID. %s", err.Error())
		c.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	_, err = c.storage.DeleteExpense(id)
	if err != nil {
		c.logger.Error("Failed to delete expense", "error", err, "id", id)

		data := expenseViewData{}
		data.Error = fmt.Sprintf("Error deleting the expense. %s", err.Error())
		c.templates.Render(w, "pages/expenses/edit.html", data)
		return
	}

	c.logger.Info("Expense deleted successfully", "id", id)

	c.resetCache()

	c.expensesHandler(w, &banner{
		Icon:    "🔥",
		Message: "Expense deleted",
	})
}

func (c *expenseHandler) expenseSearchHandler(w http.ResponseWriter, r *http.Request) {
	data := expesesViewData{}
	err := r.ParseForm()

	if err != nil {
		data.Error = err.Error()
		c.templates.Render(w, "pages/expenses/index.html", data)
		return
	}

	query := r.FormValue("q")

	if query == "" {
		data.Error = "You must provide a search criteria"
		c.templates.Render(w, "pages/expenses/index.html", data)
		return
	}

	expenses, err := c.storage.SearchExpenses(query)
	if err != nil {
		data.Error = "You must provide a search criteria"
		c.templates.Render(w, "pages/expenses/index.html", data)
	}

	groupedExpenses, years, err := expensesGroupByYearAndMonth(expenses, c.storage)
	if err != nil {
		data.Error = fmt.Sprintf("Error grouping expenses: %s", err.Error())
		c.templates.Render(w, "pages/expenses/index.html", data)
	}
	today := time.Now()

	data.Expenses = groupedExpenses
	data.Years = years
	data.Months = months
	data.CurrentYear = today.Year()
	data.CurrentMonth = today.Month().String()
	data.Query = query

	c.templates.Render(w, "pages/expenses/index.html", data)
}
