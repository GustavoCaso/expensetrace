package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/export"
	"github.com/GustavoCaso/expensetrace/internal/filter"
	pkgStorage "github.com/GustavoCaso/expensetrace/internal/storage"
)

var newAction = "new"
var editAction = "edit"

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
		c.expenseHandler(r.Context(), w, r)
	})

	mux.HandleFunc("GET /expense/new", func(w http.ResponseWriter, r *http.Request) {
		c.newExpenseHandler(r.Context(), w)
	})

	mux.HandleFunc("POST /expense", func(w http.ResponseWriter, r *http.Request) {
		c.createExpenseHandler(r.Context(), w, r)
	})

	mux.HandleFunc("PUT /expense/{id}", func(w http.ResponseWriter, r *http.Request) {
		c.updateExpenseHandler(r.Context(), w, r)
	})

	mux.HandleFunc("DELETE /expense/{id}", func(w http.ResponseWriter, r *http.Request) {
		c.deleteExpenseHandler(r.Context(), w, r)
	})

	mux.HandleFunc("GET /expenses", func(w http.ResponseWriter, r *http.Request) {
		c.expensesHandler(r.Context(), w, r, nil)
	})

	mux.HandleFunc("GET /expenses/export", func(w http.ResponseWriter, r *http.Request) {
		c.exportExpensesHandler(r.Context(), w)
	})
}

type expesesViewData struct {
	viewBase
	Expenses     expensesByYear
	Years        []int
	Months       []string
	CurrentYear  int
	CurrentMonth string
	Filter       *filter.ExpenseFilter
	Sort         *filter.SortOptions
}

func (c *expenseHandler) expensesHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, banner *banner) {
	userID := userIDFromContext(ctx)
	base := newViewBase(ctx, c.storage, c.logger, pageExpenses)
	data := expesesViewData{
		viewBase: base,
	}

	defer func() {
		c.templates.Render(w, "pages/expenses/index.html", data)
	}()

	// Parse filters from URL
	expenseFilter, sortOptions, err := filter.ParseExpenseFilters(r.URL.Query())
	if err != nil {
		data.Error = fmt.Sprintf("Invalid filters: %s", err.Error())
		return
	}

	// Get filtered expenses
	expenses, err := c.storage.GetExpensesFiltered(ctx, userID, expenseFilter, sortOptions)
	if err != nil {
		data.Error = err.Error()
		return
	}

	// Group by year and month (existing logic)
	groupedExpenses, years, err := expensesGroupByYearAndMonth(ctx, userID, expenses, c.storage)
	if err != nil {
		data.Error = fmt.Sprintf("Error grouping expenses: %s", err.Error())
		return
	}

	today := time.Now()
	data.Expenses = groupedExpenses
	data.Years = years
	data.Months = months
	data.CurrentYear = today.Year()
	data.CurrentMonth = today.Month().String()

	data.Filter = expenseFilter
	data.Sort = sortOptions

	if banner != nil {
		data.Banner = *banner
	}
}

func expensesGroupByYearAndMonth(
	ctx context.Context,
	userID int64,
	expenses []pkgStorage.Expense,
	storage pkgStorage.Storage,
) (expensesByYear, []int, error) {
	groupedExpenses := expensesByYear{}
	years := []int{}

	for _, expense := range expenses {
		var category pkgStorage.Category

		if expense.CategoryID() != nil {
			c, categoryErr := storage.GetCategory(ctx, userID, *expense.CategoryID())
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
	Action     string
	RedirectTo string
}

func (c *expenseHandler) newExpenseHandler(ctx context.Context, w http.ResponseWriter) {
	userID := userIDFromContext(ctx)
	base := newViewBase(ctx, c.storage, c.logger, pageExpenses)
	data := expenseViewData{
		viewBase: base,
	}
	data.Action = newAction

	defer func() {
		c.templates.Render(w, "pages/expenses/new.html", data)
	}()

	categories, err := c.storage.GetCategories(ctx, userID)
	if err != nil {
		c.logger.Error("Failed to get categories", "error", err)
		data.Error = fmt.Sprintf("Failed to get categories: %s", err.Error())
		return
	}

	data.Categories = categories
	data.Expense = &expenseView{
		Expense:  pkgStorage.NewExpense(0, "", "", "", 0, time.Now(), pkgStorage.ChargeType, nil),
		category: pkgStorage.NewCategory(0, "", "", 0),
	}
}

func (c *expenseHandler) createExpenseHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	base := newViewBase(ctx, c.storage, c.logger, pageExpenses)
	data := expenseViewData{
		viewBase: base,
	}
	data.Action = "new"
	data.CurrentPage = pageExpenses
	data.FormErrors = make(map[string]string)
	data.Expense = &expenseView{
		Expense:  pkgStorage.NewExpense(0, "", "", "", 0, time.Now(), pkgStorage.ChargeType, nil),
		category: pkgStorage.NewCategory(0, "", "", 0),
	}

	defer func() {
		c.templates.Render(w, "pages/expenses/new.html", data)
	}()

	categories, categoriesErr := c.storage.GetCategories(ctx, userID)
	if categoriesErr != nil {
		c.logger.Error("Failed to fetch categories", "error", categoriesErr)
		data.Error = categoriesErr.Error()
		return
	}

	data.Categories = categories
	newExpense, err := parseExpenseForm(r, w, 0, data.FormErrors)
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)

	if err != nil {
		c.logger.Error("Failed to parse form", "error", err)
		data.Error = err.Error()
		return
	}

	if len(data.FormErrors) > 0 {
		return
	}

	created, err := c.storage.InsertExpenses(ctx, userID, []pkgStorage.Expense{newExpense})
	if err != nil {
		c.logger.Error("Failed to create expense", "error", err)
		data.Error = err.Error()
		return
	}

	if created != 1 {
		c.logger.Error("Failed to create expense")
		data.Error = "Expense not created :("
		return
	}

	c.logger.Info("Expense created successfully")

	data.Banner = banner{
		Icon:    "✅",
		Message: "Expense Created",
	}
}

func (c *expenseHandler) expenseHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	base := newViewBase(ctx, c.storage, c.logger, pageExpenses)
	data := expenseViewData{
		viewBase: base,
	}
	data.Action = editAction
	data.Expense = &expenseView{
		Expense:  pkgStorage.NewExpense(0, "", "", "", 0, time.Now(), pkgStorage.ChargeType, nil),
		category: pkgStorage.NewCategory(0, "", "", 0),
	}

	defer func() {
		c.templates.Render(w, "pages/expenses/edit.html", data)
	}()

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		data.Error = err.Error()
		return
	}

	expense, err := c.storage.GetExpenseByID(ctx, userID, id)
	if err != nil {
		data.Error = err.Error()
		return
	}

	categories, err := c.storage.GetCategories(ctx, userID)
	if err != nil {
		c.logger.Error("Failed to get categories", "error", err)
		categories = []pkgStorage.Category{}
	}

	var category pkgStorage.Category
	if expense.CategoryID() != nil {
		cat, categoryErr := c.storage.GetCategory(ctx, userID, *expense.CategoryID())
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
	data.RedirectTo = r.URL.Query().Get("redirect_to")
}

func (c *expenseHandler) updateExpenseHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	base := newViewBase(ctx, c.storage, c.logger, pageExpenses)
	data := expenseViewData{
		viewBase: base,
	}
	data.FormErrors = make(map[string]string)
	data.Action = "edit"
	data.Expense = &expenseView{
		Expense:  pkgStorage.NewExpense(0, "", "", "", 0, time.Now(), pkgStorage.ChargeType, nil),
		category: pkgStorage.NewCategory(0, "", "", 0),
	}

	redirected := false
	defer func() {
		if !redirected {
			c.templates.Render(w, "pages/expenses/edit.html", data)
		}
	}()

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)

	if err != nil {
		data.Error = err.Error()
		return
	}

	categories, categoriesErr := c.storage.GetCategories(ctx, userID)
	if categoriesErr != nil {
		data.Error = categoriesErr.Error()
		return
	}
	data.Categories = categories

	expense, expenseErr := c.storage.GetExpenseByID(ctx, userID, id)
	if expenseErr != nil {
		data.Error = expenseErr.Error()
		return
	}

	var category pkgStorage.Category
	if expense.CategoryID() != nil {
		cat, categoryErr := c.storage.GetCategory(ctx, userID, *expense.CategoryID())
		if categoryErr != nil {
			c.logger.Error("Failed to get category", "error", categoryErr)
		}
		category = cat
	}

	data.Expense = &expenseView{
		Expense:  expense,
		category: category,
	}

	updatedExpense, err := parseExpenseForm(r, w, id, data.FormErrors)
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)

	if err != nil {
		c.logger.Error("Failed to parse form", "error", err)
		data.Error = err.Error()
		return
	}

	redirectTo := r.FormValue("redirect_to")

	if len(data.FormErrors) > 0 {
		data.RedirectTo = redirectTo
		return
	}

	updated, err := c.storage.UpdateExpense(ctx, userID, updatedExpense)
	if err != nil {
		c.logger.Error("Failed to update expense", "error", err, "id", id)
		data.FormErrors["failed to update expense"] = err.Error()
		data.RedirectTo = redirectTo
		return
	}

	if updated != 1 {
		c.logger.Error("Failed to update expense", "id", id)
		data.FormErrors["failed to update expense"] = "No record updated"
		data.RedirectTo = redirectTo
		return
	}

	c.logger.Info("Expense updated successfully", "id", id)

	if isValidRedirectTarget(redirectTo) {
		//nolint:canonicalheader //HTMX header
		w.Header().Set("HX-Redirect", redirectTo)
		w.WriteHeader(http.StatusNoContent)
		redirected = true
		return
	}

	var updatedCategory pkgStorage.Category
	if updatedExpense.CategoryID() != nil {
		cat, categoryErr := c.storage.GetCategory(ctx, userID, *updatedExpense.CategoryID())
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
}

// isValidRedirectTarget ensures redirect targets are relative paths only, preventing open redirects.
// Checks both raw and unescaped paths so /%2Fevil.com is rejected alongside //evil.com.
func isValidRedirectTarget(target string) bool {
	if target == "" {
		return false
	}
	u, err := url.Parse(target)
	if err != nil {
		return false
	}
	if u.Scheme != "" || u.Host != "" {
		return false
	}
	unescaped, err := url.PathUnescape(u.Path)
	if err != nil {
		return false
	}
	return len(unescaped) > 0 && unescaped[0] == '/' && (len(unescaped) < 2 || unescaped[1] != '/')
}

func parseExpenseForm(
	r *http.Request,
	w http.ResponseWriter,
	id int64,
	formErrors map[string]string,
) (pkgStorage.Expense, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}

	source := r.FormValue("source")
	description := r.FormValue("description")
	amountStr := r.FormValue("amount")
	currency := r.FormValue("currency")
	dateStr := r.FormValue("date")
	typeStr := r.FormValue("type")
	categoryIDStr := r.FormValue("category_id")

	if source == "" {
		formErrors["source"] = sourceIsRequired
	}
	if description == "" {
		formErrors["description"] = descriptionIsRequired
	}
	if currency == "" {
		formErrors["currency"] = currencyIsRequired
	}

	var amount int64
	var expenseType pkgStorage.ExpenseType

	if amountStr == "" {
		formErrors["amount"] = amountIsRequired
	} else {
		amountFloat, parseErr := strconv.ParseFloat(amountStr, 64)
		if parseErr != nil {
			formErrors["amount"] = amountInvalidFormat
		} else {
			amount = int64(amountFloat * centsMultiplier)
		}
	}

	var date time.Time
	if dateStr == "" {
		formErrors["date"] = dateIsRequired
	} else {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			formErrors["date"] = dateInvalidFormat
		}
	}

	if typeStr == "" {
		formErrors["type"] = typeIsRequired
	} else {
		typeInt, parseErr := strconv.Atoi(typeStr)
		if parseErr != nil {
			formErrors["type"] = typeInvalid
		} else {
			expenseType = pkgStorage.ExpenseType(typeInt)
		}
	}

	if amount != 0 {
		if expenseType == pkgStorage.ChargeType && amount > 0 {
			amount = -amount
		} else if expenseType == pkgStorage.IncomeType && amount < 0 {
			amount = -amount
		}
	}

	var categoryID *int64
	if categoryIDStr != "" {
		catID, parseErr := strconv.ParseInt(categoryIDStr, 10, 64)
		if parseErr != nil {
			formErrors["category_id"] = categoryInvalid
			categoryID = nil
		} else {
			categoryID = &catID
		}
	} else {
		categoryID = nil
	}

	return pkgStorage.NewExpense(id, source, description, currency, amount, date, expenseType, categoryID), nil
}

func (c *expenseHandler) deleteExpenseHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	var err error

	defer func() {
		if err != nil {
			base := newViewBase(ctx, c.storage, c.logger, pageExpenses)
			base.Error = err.Error()
			c.templates.Render(w, "pages/expenses/edit.html", base)
		}
	}()

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		err = fmt.Errorf("invalid the ID. %s", err.Error())
		return
	}

	_, err = c.storage.DeleteExpense(ctx, userID, id)
	if err != nil {
		c.logger.Error("Failed to delete expense", "error", err, "id", id)
		err = fmt.Errorf("error deleting the expense. %s", err.Error())
		return
	}

	c.logger.Info("Expense deleted successfully", "id", id)

	c.expensesHandler(ctx, w, r, &banner{
		Icon:    "🔥",
		Message: "Expense deleted",
	})
}

func (c *expenseHandler) exportExpensesHandler(ctx context.Context, w http.ResponseWriter) {
	userID := userIDFromContext(ctx)
	// Get all expenses
	expenses, err := c.storage.GetAllExpenseTypes(ctx, userID)
	if err != nil {
		c.logger.Error("Failed to get expenses for export", "error", err)
		http.Error(w, fmt.Sprintf("Failed to get expenses: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Set CSV headers
	filename := fmt.Sprintf("expenses_%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Export to CSV
	if exportErr := export.CSV(ctx, userID, w, expenses, c.storage); exportErr != nil {
		c.logger.Error("Failed to export expenses to CSV", "error", exportErr)
		http.Error(w, fmt.Sprintf("Failed to export expenses: %s", exportErr.Error()), http.StatusInternalServerError)
		return
	}

	c.logger.Info("Expenses exported successfully", "count", len(expenses))
}
