package router

import (
	"context"
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

	mux.HandleFunc("GET /category/new", func(w http.ResponseWriter, r *http.Request) {
		base := newViewBase(r.Context(), c.storage, c.logger, pageCategories)
		c.templates.Render(w, "pages/categories/new.html", base)
	})

	mux.HandleFunc("GET /category/uncategorized", func(w http.ResponseWriter, r *http.Request) {
		c.uncategorizedHandler(r.Context(), w, "", nil)
	})

	mux.HandleFunc("PUT /category/{id}", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			c.logger.Error("Failed to parse form", "error", err)

			data := struct {
				Error string
			}{
				Error: err.Error(),
			}
			c.templates.Render(w, "partials/categories/card.html", data)
			return
		}

		categoryID := r.PathValue("id")
		name := r.FormValue("name")
		pattern := r.FormValue("pattern")

		c.updatecategoryHandler(r.Context(), categoryID, name, pattern, w)
	})

	mux.HandleFunc("DELETE /category/{id}", func(w http.ResponseWriter, r *http.Request) {
		c.deletecategoryHandler(r.Context(), w, r)
	})

	mux.HandleFunc("POST /category/check", func(w http.ResponseWriter, r *http.Request) {
		c.createcategoryHandler(r.Context(), false, w, r)
	})

	mux.HandleFunc("POST /category", func(w http.ResponseWriter, r *http.Request) {
		c.createcategoryHandler(r.Context(), true, w, r)
	})

	mux.HandleFunc("POST /category/reset", func(w http.ResponseWriter, r *http.Request) {
		c.resetcategoryHandler(r.Context(), w)
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

func (c *categoryHandler) updatecategoryHandler(
	ctx context.Context,
	id, name, pattern string,
	w http.ResponseWriter,
) {
	userID := userIDFromContext(ctx)
	var categoryCardData *enhancedCategory
	var err error

	defer func() {
		if err != nil {
			errorInfo := newViewBase(ctx, c.storage, c.logger, pageCategories)
			errorInfo.Error = err.Error()
			c.templates.Render(w, "pages/categories/index.html", errorInfo)
		} else if categoryCardData != nil {
			c.templates.Render(w, "partials/categories/card.html", *categoryCardData)
		}
	}()

	categoryID, err := strconv.Atoi(id)
	if err != nil {
		return
	}

	categoryIDInt64 := int64(categoryID)

	categoryEntry, err := c.storage.GetCategory(ctx, userID, categoryIDInt64)
	if err != nil {
		return
	}

	// Get expenses that currently belong to this specific category
	currentCategoryExpenses, err := c.storage.GetExpensesByCategory(ctx, userID, categoryIDInt64)
	if err != nil {
		return
	}

	existingEnhancedCategory := createEnhancedCategory(categoryEntry, currentCategoryExpenses)

	nameChanged := name != "" && categoryEntry.Name() != name
	patternChanged := pattern != "" && categoryEntry.Pattern() != pattern

	if !nameChanged && !patternChanged {
		categoryCardData = &existingEnhancedCategory
		return
	}

	if patternChanged {
		_, err = regexp.Compile(pattern)
		if err != nil {
			existingEnhancedCategory.Errors = true
			existingEnhancedCategory.ErrorStrings = map[string]string{
				"pattern": fmt.Sprintf("invalid pattern %v", err),
			}
			categoryCardData = &existingEnhancedCategory
			return
		}
	}

	err = c.storage.UpdateCategory(ctx, userID, categoryIDInt64, name, pattern)
	if err != nil {
		existingEnhancedCategory.Errors = true
		existingEnhancedCategory.ErrorStrings = map[string]string{
			"name": fmt.Sprintf("failed to updated category %v", err),
		}
		categoryCardData = &existingEnhancedCategory
		return
	}

	c.logger.Info("Category updated successfully", "id", categoryID)

	updatedCategory := storage.NewCategory(categoryIDInt64, name, pattern)

	if !patternChanged {
		c.resetCache()
		updatedEnhancedCat := createEnhancedCategory(updatedCategory, currentCategoryExpenses)
		categoryCardData = &updatedEnhancedCat
		return
	}

	err = c.updateCategoryMatcher(userID)
	if err != nil {
		return
	}

	// Get uncategorized expenses to potentially categorize them
	uncategorizedExpenses, err := c.storage.GetExpensesWithoutCategory(ctx, userID)
	if err != nil {
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
			err = updatedErr
			return
		}

		c.logger.Info("Category's expenses updated successfully", "id", categoryID, "total", updated)

		if int(updated) != len(toUpdated) {
			c.logger.Warn("not all categories were updated")
		}

		c.resetCache()

		updatedExpenses, updateExpensesErr := c.storage.GetExpensesByCategory(ctx, userID, categoryIDInt64)
		if updateExpensesErr != nil {
			err = updateExpensesErr
			return
		}

		updatedEnhancedCat := createEnhancedCategory(categoryEntry, updatedExpenses)
		categoryCardData = &updatedEnhancedCat
		return
	}

	updatedEnhancedCat := createEnhancedCategory(updatedCategory, currentCategoryExpenses)
	categoryCardData = &updatedEnhancedCat
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

	err = c.storage.UpdateCategory(ctx, userID, cat.ID(), cat.Name(), extendedRegex)
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

		c.resetCache()
	}

	c.uncategorizedHandler(ctx, w, "", &banner{
		Icon:    "✅",
		Message: fmt.Sprintf("%d expenses categorized to %s", len(updatedExpenses), cat.Name()),
	})
}

func (c *categoryHandler) resetcategoryHandler(ctx context.Context, w http.ResponseWriter) {
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

	c.resetCache()

	c.categoriesHandler(ctx, w, nil, &banner{
		Icon:    "🔥",
		Message: "All categories deleted",
	})
}

func (c *categoryHandler) deletecategoryHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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

	c.resetCache()

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
	Name    string
	Pattern string
	Results []storage.Expense
	Total   int
}

func (c *categoryHandler) createcategoryHandler(
	ctx context.Context,
	create bool,
	w http.ResponseWriter,
	r *http.Request,
) {
	userID := userIDFromContext(ctx)
	data := createCategoryViewData{}
	data.LoggedIn = true
	data.CurrentPage = pageCategories
	template := "partials/categories/test_category.html"
	if create {
		template = "pages/categories/new.html"
	}

	defer func() {
		c.templates.Render(w, template, data)
	}()

	err := r.ParseForm()
	if err != nil {
		c.logger.Error(fmt.Sprintf("error r.ParseForm() %s", err.Error()))

		data.Error = err.Error()
		return
	}

	name := r.FormValue("name")
	pattern := r.FormValue("pattern")

	data.Name = name
	data.Pattern = pattern

	if name == "" || pattern == "" {
		data.Error =
			"category must include name and a valid regex pattern. Ensure that you populate the name and pattern input"
		return
	}

	re, err := regexp.Compile(pattern)

	if err != nil {
		data.Error = err.Error()
		return
	}

	expenses, err := c.storage.GetExpensesWithoutCategory(ctx, userID)

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

	total := len(toUpdated)

	if create && total > 0 {
		categoryID, createErr := c.storage.CreateCategory(ctx, userID, name, pattern)

		if createErr != nil {
			data.Error = createErr.Error()
			return
		}

		c.logger.Info("Category created", "name", name, "pattern", pattern)

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
			data.Error = updateErr.Error()
			return
		}

		c.logger.Info("Category expenses updated", "total", updated)

		if int(updated) != total {
			c.logger.Warn("not all categories were updated")

			total = int(updated)
		}

		updateCategoryMatcherErr := c.updateCategoryMatcher(userID)
		if updateCategoryMatcherErr != nil {
			data.Error = updateCategoryMatcherErr.Error()
			return
		}

		c.resetCache()
		data.Banner = banner{
			Icon: "✅",
			Message: fmt.Sprintf(
				"Category %s was created successfully and %d transactions were categorized!",
				name,
				total,
			),
		}
		return
	}

	data.Total = total
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
