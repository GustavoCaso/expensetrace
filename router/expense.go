package router

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
)

var newAction = "new"
var editAction = "edit"

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

func (c *expenseHandler) expensesHandler(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	banner *domain.Banner,
) {
	userID := userIDFromContext(ctx)
	base := viewBaseFromContext(ctx)
	data := domain.ExpensesViewData{
		ViewBase: base,
		Filter:   &domain.ExpenseFilter{},
		Sort:     &domain.SortOptions{},
	}

	defer func() {
		c.renderHTML(w, http.StatusOK, data, "base", "pages/expenses/index.html")
	}()

	// Parse filters from URL
	expenseFilter, sortOptions, err := domain.ParseExpenseFilters(r.URL.Query())
	if err != nil {
		data.Error = fmt.Sprintf("Invalid filters: %s", err.Error())
		return
	}

	// Get filtered expenses
	expenses, err := c.expenseService.List(ctx, userID, expenseFilter, sortOptions)
	if err != nil {
		data.Error = err.Error()
		return
	}

	// Group by year and month (existing logic)
	groupedExpenses, years, err := c.expenseService.GroupByYearAndMonth(ctx, userID, expenses)
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

func (c *expenseHandler) newExpenseHandler(ctx context.Context, w http.ResponseWriter) {
	userID := userIDFromContext(ctx)
	base := viewBaseFromContext(ctx)
	data := domain.ExpenseViewData{
		ViewBase: base,
	}
	data.Action = newAction

	defer func() {
		c.renderHTML(w, http.StatusOK, data, "base", "pages/expenses/new.html")
	}()

	categories, err := c.categoryService.List(ctx, userID)
	if err != nil {
		c.logger.Error("Failed to get categories", "error", err)
		data.Error = fmt.Sprintf("Failed to get categories: %s", err.Error())
		return
	}

	data.Categories = categories
	data.Expense = &domain.ExpenseView{
		Expense: domain.NewExpense(0, "", "", "", 0, time.Now(), domain.ChargeType, nil),
		Cat:     domain.NewCategory(0, "", "", 0),
	}
}

func (c *expenseHandler) createExpenseHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	base := viewBaseFromContext(ctx)
	data := domain.ExpenseViewData{
		ViewBase: base,
	}
	data.Action = "new"
	data.CurrentPage = pageExpenses
	data.FormErrors = make(map[string]string)
	data.Expense = &domain.ExpenseView{
		Expense: domain.NewExpense(0, "", "", "", 0, time.Now(), domain.ChargeType, nil),
		Cat:     domain.NewCategory(0, "", "", 0),
	}

	defer func() {
		c.renderHTML(w, http.StatusOK, data, "base", "pages/expenses/new.html")
	}()

	categories, categoriesErr := c.categoryService.List(ctx, userID)
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

	_, err = c.expenseService.Create(ctx, userID, newExpense)
	if err != nil {
		c.logger.Error("Failed to create expense", "error", err)
		data.Error = err.Error()
		return
	}

	c.logger.Info("Expense created successfully")

	data.Banner = domain.Banner{
		Icon:    "✅",
		Message: "Expense Created",
	}
}

func (c *expenseHandler) expenseHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	base := viewBaseFromContext(ctx)
	data := domain.ExpenseViewData{
		ViewBase: base,
	}
	data.Action = editAction
	data.Expense = &domain.ExpenseView{
		Expense: domain.NewExpense(0, "", "", "", 0, time.Now(), domain.ChargeType, nil),
		Cat:     domain.NewCategory(0, "", "", 0),
	}

	defer func() {
		c.renderHTML(w, http.StatusOK, data, "base", "pages/expenses/edit.html")
	}()

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		data.Error = err.Error()
		return
	}

	expenseView, err := c.expenseService.Get(ctx, userID, id)
	if err != nil {
		data.Error = err.Error()
		return
	}

	categories, err := c.categoryService.List(ctx, userID)
	if err != nil {
		c.logger.Error("Failed to get categories", "error", err)
		categories = []domain.Category{}
	}

	data.Expense = expenseView
	data.Categories = categories
	data.RedirectTo = r.URL.Query().Get("redirect_to")
}

func (c *expenseHandler) updateExpenseHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	base := viewBaseFromContext(ctx)
	data := domain.ExpenseViewData{
		ViewBase: base,
	}
	data.FormErrors = make(map[string]string)
	data.Action = "edit"
	data.Expense = &domain.ExpenseView{
		Expense: domain.NewExpense(0, "", "", "", 0, time.Now(), domain.ChargeType, nil),
		Cat:     domain.NewCategory(0, "", "", 0),
	}

	redirected := false
	defer func() {
		if !redirected {
			c.renderHTML(w, http.StatusOK, data, "base", "pages/expenses/edit.html")
		}
	}()

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)

	if err != nil {
		data.Error = err.Error()
		return
	}

	categories, categoriesErr := c.categoryService.List(ctx, userID)
	if categoriesErr != nil {
		data.Error = categoriesErr.Error()
		return
	}
	data.Categories = categories

	expenseView, expenseErr := c.expenseService.Get(ctx, userID, id)
	if expenseErr != nil {
		data.Error = expenseErr.Error()
		return
	}

	data.Expense = expenseView

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

	updated, err := c.expenseService.Update(ctx, userID, updatedExpense)
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

	updateCategory, _ := strconv.ParseBool(r.FormValue("update_category"))

	if updateCategory {
		if updatedExpense.CategoryID() == nil {
			c.logger.Info("Expense update tried to update category but expense has no category", "id", id)
		} else {
			catgegoryUpdateError := c.categoryService.UpdateCategoryPattern(
				ctx,
				userID,
				*updatedExpense.CategoryID(),
				updatedExpense.Description(),
			)

			if catgegoryUpdateError != nil {
				c.logger.Error("Failed to update expense's category", "id", id)
				data.FormErrors["failed to update expense's catgeory"] = catgegoryUpdateError.Error()
				data.RedirectTo = redirectTo
				return
			}
		}
	}

	if isValidRedirectTarget(redirectTo) {
		w.Header().Set("Hx-Redirect", redirectTo)
		w.WriteHeader(http.StatusNoContent)
		redirected = true
		return
	}

	updatedCategory := domain.EmptyCategory()
	if updatedExpense.CategoryID() != nil {
		cat, categoryErr := c.categoryService.Get(ctx, userID, *updatedExpense.CategoryID())
		if categoryErr != nil {
			c.logger.Error("Failed to get category", "error", categoryErr)
		}
		updatedCategory = cat
	}

	data.Expense = &domain.ExpenseView{
		Expense: updatedExpense,
		Cat:     updatedCategory,
	}

	data.Banner = domain.Banner{
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
) (domain.Expense, error) {
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
	var expenseType domain.ExpenseType

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
			expenseType = domain.ExpenseType(typeInt)
		}
	}

	if amount != 0 {
		if expenseType == domain.ChargeType && amount > 0 {
			amount = -amount
		} else if expenseType == domain.IncomeType && amount < 0 {
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

	return domain.NewExpense(id, source, description, currency, amount, date, expenseType, categoryID), nil
}

func (c *expenseHandler) deleteExpenseHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	var err error

	defer func() {
		if err != nil {
			base := viewBaseFromContext(ctx)
			base.Error = err.Error()
			c.renderHTML(w, http.StatusOK, base, "base", "pages/expenses/edit.html")
		}
	}()

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		err = fmt.Errorf("invalid the ID. %s", err.Error())
		return
	}

	err = c.expenseService.Delete(ctx, userID, id)
	if err != nil {
		c.logger.Error("Failed to delete expense", "error", err, "id", id)
		err = fmt.Errorf("error deleting the expense. %s", err.Error())
		return
	}

	c.logger.Info("Expense deleted successfully", "id", id)

	c.expensesHandler(ctx, w, r, &domain.Banner{
		Icon:    "🔥",
		Message: "Expense deleted",
	})
}

func (c *expenseHandler) exportExpensesHandler(ctx context.Context, w http.ResponseWriter) {
	userID := userIDFromContext(ctx)

	// Set CSV headers
	filename := fmt.Sprintf("expenses_%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Export to CSV
	if exportErr := c.expenseService.Export(ctx, userID, w); exportErr != nil {
		c.logger.Error("Failed to export expenses to CSV", "error", exportErr)
		http.Error(w, fmt.Sprintf("Failed to export expenses: %s", exportErr.Error()), http.StatusInternalServerError)
		return
	}

	c.logger.Info("Expenses exported successfully")
}
