package router

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

const errSearchCriteria = "You must provide a search criteria"

type categoryHandler struct {
	*router
}

func (c *categoryHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /categories", func(w http.ResponseWriter, r *http.Request) {
		c.categoriesHandler(r.Context(), w, nil, nil)
	})

	mux.HandleFunc("GET /category/{id}", func(w http.ResponseWriter, r *http.Request) {
		c.categoryHandler(r.Context(), w, r)
	})

	mux.HandleFunc("GET /category/new", func(w http.ResponseWriter, r *http.Request) {
		base := newViewBase(r.Context(), c.storage, c.logger, pageCategories)
		data := categoryViewData{
			viewBase: base,
			Action:   newAction,
			Category: storage.EmptyCategory(),
		}
		c.templates.Render(w, "pages/categories/new.html", data)
	})

	mux.HandleFunc("GET /category/uncategorized", func(w http.ResponseWriter, r *http.Request) {
		c.uncategorizedHandler(r.Context(), w, "", nil)
	})

	mux.HandleFunc("PUT /category/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		c.updateCategoryHandler(ctx, w, r)
	})

	mux.HandleFunc("DELETE /category/{id}", func(w http.ResponseWriter, r *http.Request) {
		c.deleteCategoryHandler(r.Context(), w, r)
	})

	mux.HandleFunc("POST /category/test", func(w http.ResponseWriter, r *http.Request) {
		c.testCategoryHandler(r.Context(), w, r)
	})

	mux.HandleFunc("POST /category", func(w http.ResponseWriter, r *http.Request) {
		c.createCategoryHandler(r.Context(), w, r)
	})

	mux.HandleFunc("POST /category/reset", func(w http.ResponseWriter, r *http.Request) {
		c.resetCategoryHandler(r.Context(), w)
	})

	mux.HandleFunc("POST /category/uncategorized/update", func(w http.ResponseWriter, r *http.Request) {
		c.updateUncategorizedHandler(r.Context(), w, r)
	})

	mux.HandleFunc("POST /category/uncategorized/search", func(w http.ResponseWriter, r *http.Request) {
		data := viewBase{}
		err := r.ParseForm()

		if err != nil {
			data.Error = err.Error()
			c.templates.Render(w, "pages/categories/uncategorized.html", data)
			return
		}

		query := r.FormValue("q")

		if query == "" {
			data.Error = errSearchCriteria
			c.templates.Render(w, "pages/categories/uncategorized.html", data)
			return
		}

		c.uncategorizedHandler(r.Context(), w, query, nil)
	})
}

// enhancedCategory extends storage.Category with extra UI-friendly fields.
type enhancedCategory struct {
	storage.Category
	AvgAmount       int64
	LastTransaction string
	Total           int
	TotalAmount     int64
	SpendingCount   int
	IncomeCount     int
	Errors          bool
	ErrorStrings    map[string]string
}

// categoryFormData holds parsed and validated category form data.
type categoryFormData struct {
	Name          string
	Pattern       string
	MonthlyBudget int64
}

type categoriesViewData struct {
	viewBase
	Categories         []enhancedCategory
	CategorizedCount   int
	UncategorizedCount int
}

func (c *categoryHandler) categoriesHandler(
	ctx context.Context,
	w http.ResponseWriter,
	outerErr error,
	banner *banner,
) {
	userID := userIDFromContext(ctx)
	base := newViewBase(ctx, c.storage, c.logger, pageCategories)
	data := categoriesViewData{
		viewBase: base,
	}

	defer func() {
		c.templates.Render(w, "pages/categories/index.html", data)
	}()

	categories, err := c.storage.GetCategories(ctx, userID)
	if err != nil {
		data.Error = fmt.Sprintf("error fetch categories: %s", err.Error())
		return
	}

	categoriesWithoutExclude := []storage.Category{}
	for _, category := range categories {
		if category.Name() == storage.ExcludeCategory {
			continue
		}
		categoriesWithoutExclude = append(categoriesWithoutExclude, category)
	}

	// Get counts for uncategorized expenses
	uncategorizedInfos, err := c.storage.GetExpensesWithoutCategory(ctx, userID)
	if err != nil {
		data.Error = err.Error()
		return
	}
	uncategorizedCount := len(uncategorizedInfos)

	// Get total categorized count
	totalCategorized := 0

	// Enhance categories with additional data
	enhancedCategories := make([]enhancedCategory, len(categoriesWithoutExclude))

	for i, cat := range categoriesWithoutExclude {
		// Get expenses for this category
		expenses, expensesErr := c.storage.GetExpensesByCategory(ctx, userID, cat.ID())
		if expensesErr != nil {
			data.Error = expensesErr.Error()
			return
		}

		totalCategorized += len(expenses)
		enhancedCategories[i] = createEnhancedCategory(cat, expenses)
	}

	data.Categories = enhancedCategories
	data.CategorizedCount = totalCategorized
	data.UncategorizedCount = uncategorizedCount

	if outerErr != nil {
		data.Error = outerErr.Error()
	}

	if banner != nil {
		data.Banner = *banner
	}
}

type categoryViewData struct {
	viewBase
	Category storage.Category
	Action   string
}

func (c *categoryHandler) categoryHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	base := newViewBase(ctx, c.storage, c.logger, pageCategories)
	data := categoryViewData{
		viewBase: base,
		Action:   editAction,
	}
	var err error
	defer func() {
		if err != nil {
			data.Error = err.Error()
		}
		c.templates.Render(w, "pages/categories/edit.html", data)
	}()

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return
	}

	categoryEntry, err := c.storage.GetCategory(ctx, userID, id)
	if err != nil {
		return
	}

	data.Category = categoryEntry
}

func (c *categoryHandler) updateCategoryHandler(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	userID := userIDFromContext(ctx)
	var err error
	base := newViewBase(ctx, c.storage, c.logger, pageCategories)
	data := categoryViewData{
		viewBase: base,
		Action:   editAction,
	}
	data.Category = storage.EmptyCategory()

	defer func() {
		if err != nil {
			data.Error = err.Error()
		}
		c.templates.Render(w, "pages/categories/edit.html", data)
	}()

	categoryID := r.PathValue("id")

	categoryIDInt64, parseErr := strconv.ParseInt(categoryID, 10, 64)
	if parseErr != nil {
		c.logger.Error("Invalid category ID", "error", parseErr, "id", categoryID)
		return
	}

	existingCategory, err := c.storage.GetCategory(ctx, userID, categoryIDInt64)
	if err != nil {
		c.logger.Error("Failed to get category", "error", err, "id", categoryID)
		return
	}

	// Parse form manually for partial updates
	if formErr := r.ParseForm(); formErr != nil {
		c.logger.Error("Failed to parse form", "error", formErr)
		data.Error = fmt.Errorf("invalid form data: %w", formErr).Error()
		return
	}

	// Get form values, defaulting to existing values if not provided
	name := r.FormValue("name")
	if name == "" {
		name = existingCategory.Name()
	}

	pattern := r.FormValue("pattern")
	if pattern == "" {
		pattern = existingCategory.Pattern()
	}

	budgetStr := r.FormValue("monthly_budget")
	monthlyBudget := existingCategory.MonthlyBudget()
	if budgetStr != "" {
		var budgetErr error
		monthlyBudget, budgetErr = validateBudget(budgetStr)
		if budgetErr != nil {
			c.logger.Error("Failed to validate budget", "error", budgetErr)
			data.Error = budgetErr.Error()
			return
		}
	}

	// Validate pattern
	if _, compileErr := regexp.Compile(pattern); compileErr != nil {
		c.logger.Error("Invalid pattern", "error", compileErr)
		data.Error = fmt.Errorf("invalid pattern: %w", compileErr).Error()
		return
	}

	nameChanged := existingCategory.Name() != name
	patternChanged := existingCategory.Pattern() != pattern
	budgetChanged := !budgetsEqual(monthlyBudget, existingCategory.MonthlyBudget())

	if !nameChanged && !patternChanged && !budgetChanged {
		data.Banner = banner{
			Message: "Nothing to update",
		}
		return
	}

	err = c.storage.UpdateCategory(
		ctx,
		userID,
		categoryIDInt64,
		name,
		pattern,
		monthlyBudget,
	)
	if err != nil {
		c.logger.Error("Failed to update category", "error", err)
		err = fmt.Errorf("failed to update category: %w", err)
		return
	}

	c.logger.Info("Category updated successfully", "id", categoryID)

	updatedCategory, err := c.storage.GetCategory(ctx, userID, categoryIDInt64)
	if err != nil {
		c.logger.Error("Failed to get updated category", "error", err)
		err = fmt.Errorf("failed to get updated category: %w", err)
		return
	}

	data.Category = updatedCategory

	data.Banner = banner{
		Icon:    "✅",
		Message: "Category Updated",
	}

	if !patternChanged {
		return
	}

	err = c.updateCategoryMatcher(userID)
	if err != nil {
		return
	}

	// Get uncategorized expenses to potentially categorize them
	uncategorizedExpenses, err := c.storage.GetExpensesWithoutCategory(ctx, userID)
	if err != nil {
		c.logger.Error("Failed to get uncategorized expenses", "error", err)
		return
	}

	currentCategoryExpenses, err := c.storage.GetExpensesByCategory(ctx, userID, categoryIDInt64)
	if err != nil {
		c.logger.Error("Failed to get cateory expenses", "error", err)
		return
	}

	// Combine both sets of expenses to process
	expensesToProcess := make([]storage.Expense, 0, len(currentCategoryExpenses)+len(uncategorizedExpenses))
	expensesToProcess = append(expensesToProcess, currentCategoryExpenses...)
	expensesToProcess = append(expensesToProcess, uncategorizedExpenses...)
	toUpdated := []storage.Expense{}

	for _, ex := range expensesToProcess {
		id, _ := c.matcher.Match(ex.Description())

		// 1. match && expense does not have a category OR the existing category is different
		// 2. no match && expense is part of the category we are updating
		if (id != nil && (ex.CategoryID() == nil || *ex.CategoryID() != *id)) ||
			(id == nil && expenseBelongsToCategoryWeAreUpdating(ex, categoryIDInt64)) {
			expense := storage.NewExpense(
				ex.ID(),
				ex.Source(),
				ex.Description(),
				ex.Currency(),
				ex.Amount(),
				ex.Date(),
				ex.Type(),
				id,
			)
			toUpdated = append(toUpdated, expense)
		}
	}

	if len(toUpdated) > 0 {
		updated, updatedErr := c.storage.UpdateExpenses(ctx, userID, toUpdated)
		if updatedErr != nil {
			c.logger.Error("Failed to get update expenses", "error", err)
			err = updatedErr
			return
		}

		c.logger.Info("Category's expenses updated successfully", "id", categoryID, "total", updated)

		if int(updated) != len(toUpdated) {
			c.logger.Warn("not all categories were updated")
		}
	}
}

func expenseBelongsToCategoryWeAreUpdating(ex storage.Expense, categoryID int64) bool {
	return ex.CategoryID() != nil && *ex.CategoryID() == categoryID
}

type uncategorizedInfo struct {
	Count    int
	Expenses []storage.Expense
	Total    int64
	Slug     string
}

type uncategorizedViewData struct {
	viewBase
	Keys             []string
	UncategorizeInfo map[string]uncategorizedInfo
	Categories       []storage.Category
	TotalExpenses    int
	TotalAmount      int64
}

func (c *categoryHandler) uncategorizedHandler(
	ctx context.Context,
	w http.ResponseWriter,
	query string,
	banner *banner,
) {
	userID := userIDFromContext(ctx)
	base := newViewBase(ctx, c.storage, c.logger, pageCategories)
	data := uncategorizedViewData{
		viewBase: base,
	}

	defer func() {
		c.templates.Render(w, "pages/categories/uncategorized.html", data)
	}()

	var expenses []storage.Expense
	var err error

	if query != "" {
		expenses, err = c.storage.GetExpensesWithoutCategoryWithQuery(ctx, userID, query)
	} else {
		expenses, err = c.storage.GetExpensesWithoutCategory(ctx, userID)
	}

	if err != nil {
		data.Error = err.Error()
		return
	}

	uncategorizeInfo := map[string]uncategorizedInfo{}
	totalExpenses := 0
	var totalAmount int64

	for _, ex := range expenses {
		if r, ok := uncategorizeInfo[ex.Description()]; ok {
			r.Count++
			r.Expenses = append(r.Expenses, ex)
			r.Total += ex.Amount()
			uncategorizeInfo[ex.Description()] = r
		} else {
			uncategorizeInfo[ex.Description()] = uncategorizedInfo{
				Count:    1,
				Total:    ex.Amount(),
				Expenses: []storage.Expense{ex},
				Slug:     slugify(ex.Description()),
			}
		}

		totalExpenses++
		totalAmount += ex.Amount()
	}

	keys := slices.Collect(maps.Keys(uncategorizeInfo))

	sort.SliceStable(keys, func(i, j int) bool {
		return uncategorizeInfo[keys[i]].Count > uncategorizeInfo[keys[j]].Count
	})

	for _, report := range uncategorizeInfo {
		sort.SliceStable(report.Expenses, func(i, j int) bool {
			return report.Expenses[i].Date().After(report.Expenses[j].Date())
		})
	}

	data.Keys = keys
	data.UncategorizeInfo = uncategorizeInfo
	data.Categories = c.matcher.Categories()
	data.TotalExpenses = totalExpenses
	data.TotalAmount = totalAmount

	if banner != nil {
		data.Banner = *banner
	}
}

var specialCharactersRegex = regexp.MustCompile(`[^a-z0-9\-]`)
var multipleHyphenRegex = regexp.MustCompile(`[^a-z0-9\-]`)

func slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)
	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	// Remove special characters
	s = specialCharactersRegex.ReplaceAllString(s, "")
	// Replace multiple hyphens with a single one
	s = multipleHyphenRegex.ReplaceAllString(s, "-")
	// Remove leading and trailing hyphens
	s = strings.Trim(s, "-")
	return s
}

func (c *categoryHandler) updateUncategorizedHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	data := newViewBase(ctx, c.storage, c.logger, pageCategories)

	defer func() {
		if data.Error != "" {
			c.templates.Render(w, "pages/categories/uncategorized.html", data)
		}
	}()

	err := r.ParseForm()
	if err != nil {
		c.logger.Error(fmt.Sprintf("error r.ParseForm() %s", err.Error()))

		data.Error = err.Error()
		return
	}

	expenseDescription := r.FormValue("description")
	categoryIDStr := r.FormValue("category_id")

	categoryID, err := strconv.Atoi(categoryIDStr)

	if err != nil {
		c.logger.Error(fmt.Sprintf("error strconv.Atoi with value %s. %s", categoryIDStr, err.Error()))

		data.Error = err.Error()
		return
	}

	cat, err := c.storage.GetCategory(ctx, userID, int64(categoryID))

	if err != nil {
		c.logger.Error(fmt.Sprintf("error GetCategory %s", err.Error()))

		data.Error = err.Error()
		return
	}

	extendedRegex, err := extendRegex(cat.Pattern(), expenseDescription)

	if err != nil {
		c.logger.Error(fmt.Sprintf("error extendRegex %s", err.Error()))
		data.Error = err.Error()
		return
	}

	err = c.storage.UpdateCategory(ctx, userID, cat.ID(), cat.Name(), extendedRegex, cat.MonthlyBudget())
	if err != nil {
		c.logger.Error(fmt.Sprintf("error UpdateCategory %s", err.Error()))
		data.Error = err.Error()
		return
	}

	c.logger.Info("Category updated successfully", "id", cat.ID(), "extended_regex", extendedRegex)

	expenses, err := c.storage.SearchExpensesByDescription(ctx, userID, expenseDescription)

	if err != nil {
		c.logger.Error(fmt.Sprintf("error SearchExpensesByDescription %s", err.Error()))

		data.Error = err.Error()
		return
	}

	updatedExpenses := make([]storage.Expense, len(expenses))

	if len(expenses) > 0 {
		categoryID := int64(categoryID)
		for i, ex := range expenses {
			expense := storage.NewExpense(
				ex.ID(),
				ex.Source(),
				ex.Description(),
				ex.Currency(),
				ex.Amount(),
				ex.Date(),
				ex.Type(),
				&categoryID,
			)
			updatedExpenses[i] = expense
		}
		updated, updateErr := c.storage.UpdateExpenses(ctx, userID, updatedExpenses)
		if updateErr != nil {
			data.Error = updateErr.Error()
			return
		}

		c.logger.Info("Category's expenses updated successfully", "id", cat.ID(), "total", updated)

		if updated != int64(len(expenses)) {
			c.logger.Warn("not all expenses updated succesfully")
		}

		updateCategoryMatcherErr := c.updateCategoryMatcher(userID)
		if updateCategoryMatcherErr != nil {
			data.Error = updateCategoryMatcherErr.Error()
			return
		}
	}

	c.uncategorizedHandler(ctx, w, "", &banner{
		Icon:    "✅",
		Message: fmt.Sprintf("%d expenses categorized to %s", len(updatedExpenses), cat.Name()),
	})
}

func (c *categoryHandler) resetCategoryHandler(ctx context.Context, w http.ResponseWriter) {
	userID := userIDFromContext(ctx)
	_, err := c.storage.DeleteCategories(ctx, userID)

	if err != nil {
		c.categoryIndexError(ctx, w, err)
		return
	}

	updateCategoryMatcherErr := c.updateCategoryMatcher(userID)
	if updateCategoryMatcherErr != nil {
		c.categoryIndexError(ctx, w, updateCategoryMatcherErr)
		return
	}

	c.categoriesHandler(ctx, w, nil, &banner{
		Icon:    "🔥",
		Message: "All categories deleted",
	})
}

func (c *categoryHandler) deleteCategoryHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.categoriesHandler(ctx, w, fmt.Errorf("invalid ID. %s", err.Error()), nil)
		return
	}

	_, err = c.storage.DeleteCategory(ctx, userID, id)
	if err != nil {
		c.logger.Error("Failed to delete category", "error", err, "id", id)

		c.categoriesHandler(ctx, w, fmt.Errorf("error deleting the category. %s", err.Error()), nil)
		return
	}

	c.logger.Info("Category deleted successfully", "id", id)

	c.categoriesHandler(ctx, w, nil, &banner{
		Icon:    "✅",
		Message: "Category  deleted",
	})
}

func extendRegex(pattern, description string) (string, error) {
	extendedPattern := fmt.Sprintf("%s|%s", pattern, regexp.QuoteMeta(description))
	re, err := regexp.Compile(extendedPattern)
	if err != nil {
		return "", err
	}
	return re.String(), nil
}

type createCategoryViewData struct {
	viewBase
	Category storage.Category
	Results  []storage.Expense
	Total    int
	Action   string
}

func (c *categoryHandler) createCategoryHandler(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	userID := userIDFromContext(ctx)
	data := createCategoryViewData{}
	data.LoggedIn = true
	data.CurrentPage = pageCategories
	data.Action = newAction

	defer func() {
		c.templates.Render(w, "pages/categories/new.html", data)
	}()

	// Initialize with empty category
	data.Category = storage.EmptyCategory()

	// Parse and validate form using helper
	formData, err := parseCategoryForm(r)
	if err != nil {
		c.logger.Error("Failed to parse category form", "error", err)
		data.Error = err.Error()
		return
	}

	// Store parsed values in category for re-rendering
	data.Category = storage.NewCategory(0, formData.Name, formData.Pattern, formData.MonthlyBudget)

	expenses, err := c.storage.GetExpensesWithoutCategory(ctx, userID)
	if err != nil {
		c.logger.Error("Failed to get expenses without category", "error", err)
		data.Error = err.Error()
		return
	}

	re, err := regexp.Compile(formData.Pattern)
	if err != nil {
		data.Error = err.Error()
		return
	}

	toUpdated := []storage.Expense{}

	for _, ex := range expenses {
		if re.MatchString(ex.Description()) {
			toUpdated = append(toUpdated, ex)
		}
	}

	totalExpensesToUpdate := len(toUpdated)

	categoryID, createErr := c.storage.CreateCategory(
		ctx,
		userID,
		formData.Name,
		formData.Pattern,
		formData.MonthlyBudget,
	)

	if createErr != nil {
		c.logger.Error("Failed to create category", "error", createErr)
		data.Error = createErr.Error()
		return
	}

	c.logger.Info("Category created", "name", formData.Name, "pattern", formData.Pattern)

	updateCategoryMatcherErr := c.updateCategoryMatcher(userID)
	if updateCategoryMatcherErr != nil {
		c.logger.Error("Failed to update category matcher", "error", updateCategoryMatcherErr)
		data.Error = updateCategoryMatcherErr.Error()
		return
	}

	if totalExpensesToUpdate > 0 {
		updatedExpenses := make([]storage.Expense, len(toUpdated))

		for i, ex := range toUpdated {
			expense := storage.NewExpense(
				ex.ID(),
				ex.Source(),
				ex.Description(),
				ex.Currency(),
				ex.Amount(),
				ex.Date(),
				ex.Type(),
				&categoryID,
			)
			updatedExpenses[i] = expense
		}

		updated, updateErr := c.storage.UpdateExpenses(ctx, userID, updatedExpenses)
		if updateErr != nil {
			c.logger.Error("Failed to update expenses", "error", updateErr)
			data.Error = updateErr.Error()
			return
		}

		c.logger.Info("Category expenses updated", "total", updated)

		if int(updated) != totalExpensesToUpdate {
			c.logger.Warn("not all categories were updated")

			totalExpensesToUpdate = int(updated)
		}
	}

	data.Banner = banner{
		Icon: "✅",
		Message: fmt.Sprintf(
			"Category %s was created successfully and %d transactions were categorized!",
			formData.Name,
			totalExpensesToUpdate,
		),
	}

	data.Total = totalExpensesToUpdate
	data.Results = toUpdated
}

func (c *categoryHandler) testCategoryHandler(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	userID := userIDFromContext(ctx)
	data := createCategoryViewData{}
	data.LoggedIn = true
	data.CurrentPage = pageCategories

	defer func() {
		c.templates.Render(w, "partials/categories/test_category.html", data)
	}()

	// Parse and validate form using helper
	formData, err := parseCategoryForm(r)
	if err != nil {
		c.logger.Error("Failed to parse category form", "error", err)
		data.Error = err.Error()
		return
	}

	// Store parsed values in category for re-rendering
	data.Category = storage.NewCategory(0, formData.Name, formData.Pattern, formData.MonthlyBudget)

	expenses, err := c.storage.GetExpensesWithoutCategory(ctx, userID)
	if err != nil {
		c.logger.Error("Failed to get expenses without category", "error", err)
		data.Error = err.Error()
		return
	}

	re, err := regexp.Compile(formData.Pattern)
	if err != nil {
		data.Error = err.Error()
		return
	}

	toUpdated := []storage.Expense{}

	for _, ex := range expenses {
		if re.MatchString(ex.Description()) {
			toUpdated = append(toUpdated, ex)
		}
	}

	data.Total = len(toUpdated)
	data.Results = toUpdated
}

func createEnhancedCategory(category storage.Category, expenses []storage.Expense) enhancedCategory {
	var totalAmount int64
	var lastTransaction time.Time
	spendingCount := 0
	incomeCount := 0

	for _, exp := range expenses {
		totalAmount += exp.Amount()

		if exp.Amount() < 0 {
			spendingCount++
		} else {
			incomeCount++
		}

		if lastTransaction.IsZero() || exp.Date().After(lastTransaction) {
			lastTransaction = exp.Date()
		}
	}

	avgAmount := int64(0)
	if len(expenses) > 0 {
		avgAmount = totalAmount / int64(len(expenses))
	}

	lastTransactionStr := ""
	if !lastTransaction.IsZero() {
		lastTransactionStr = lastTransaction.Format("2006-01-02")
	}

	return enhancedCategory{
		Category:        category,
		AvgAmount:       avgAmount,
		LastTransaction: lastTransactionStr,
		Total:           len(expenses),
		TotalAmount:     totalAmount,
		SpendingCount:   spendingCount,
		IncomeCount:     incomeCount,
	}
}

func (c *categoryHandler) categoryIndexError(ctx context.Context, w http.ResponseWriter, err error) {
	data := newViewBase(ctx, c.storage, c.logger, pageCategories)
	data.Error = err.Error()
	c.templates.Render(w, "pages/categories/index.html", data)
}

func (c *categoryHandler) updateCategoryMatcher(userID int64) error {
	categories, categoryErr := c.storage.GetCategories(context.Background(), userID)
	if categoryErr != nil {
		return categoryErr
	}

	matcher := matcher.New(categories)
	c.matcher = matcher
	return nil
}

func validateBudget(budgetStr string) (int64, error) {
	if budgetStr == "" {
		return 0, nil // No budget
	}

	budgetFloat, err := strconv.ParseFloat(budgetStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid budget format: %w", err)
	}

	if budgetFloat < 0 {
		return 0, errors.New("budget cannot be negative")
	}

	budgetCents := int64(budgetFloat * 100) //nolint:mnd // the value is obvious
	return budgetCents, nil
}

// parseCategoryForm parses and validates category form fields from the request.
// Returns parsed data or an error with a user-friendly message.
func parseCategoryForm(r *http.Request) (*categoryFormData, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("invalid form data: %w", err)
	}

	name := r.FormValue("name")
	pattern := r.FormValue("pattern")
	budgetStr := r.FormValue("monthly_budget")

	// Validate required fields
	if name == "" || pattern == "" {
		return nil, errors.New("category must include name and a valid regex pattern")
	}

	// Validate pattern regex
	if _, err := regexp.Compile(pattern); err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	// Validate budget
	monthlyBudget, budgetErr := validateBudget(budgetStr)
	if budgetErr != nil {
		return nil, budgetErr
	}

	return &categoryFormData{
		Name:          name,
		Pattern:       pattern,
		MonthlyBudget: monthlyBudget,
	}, nil
}

func budgetsEqual(a, b int64) bool {
	return a == b
}
