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

	mux.HandleFunc("GET /category/new", func(w http.ResponseWriter, _ *http.Request) {
		c.templates.Render(w, "pages/categories/new.html", viewBase{CurrentPage: pageCategories})
	})

	mux.HandleFunc("GET /category/uncategorized", func(w http.ResponseWriter, r *http.Request) {
		c.uncategorizedHandler(r.Context(), w, "")
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
		// Category type is no longer needed - we only handle expenses
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

		c.uncategorizedHandler(r.Context(), w, query)
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
	categories, err := c.storage.GetCategories(ctx)

	if err != nil {
		c.categoryIndexError(w, fmt.Errorf("error fetch categories: %s", err.Error()))
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
	uncategorizedInfos, err := c.storage.GetExpensesWithoutCategory(ctx)
	if err != nil {
		c.categoryIndexError(w, err)
		return
	}
	uncategorizedCount := len(uncategorizedInfos)

	// Get total categorized count
	totalCategorized := 0

	// Enhance categories with additional data
	enhancedCategories := make([]enhancedCategory, len(categoriesWithoutExclude))

	for i, cat := range categoriesWithoutExclude {
		// Get expenses for this category
		expenses, expensesErr := c.storage.GetExpensesByCategory(ctx, cat.ID())

		if expensesErr != nil {
			c.categoryIndexError(w, expensesErr)
			return
		}

		totalCategorized += len(expenses)

		enhancedCategories[i] = createEnhancedCategory(cat, expenses)
	}

	data := categoriesViewData{
		Categories:         enhancedCategories,
		CategorizedCount:   totalCategorized,
		UncategorizedCount: uncategorizedCount,
	}
	data.CurrentPage = pageCategories

	if outerErr != nil {
		data.Error = outerErr.Error()
	}

	if banner != nil {
		data.Banner = *banner
	}

	c.templates.Render(w, "pages/categories/index.html", data)
}

func (c *categoryHandler) updatecategoryHandler(
	ctx context.Context,
	id, name, pattern string,
	w http.ResponseWriter,
) {
	categoryID, err := strconv.Atoi(id)

	if err != nil {
		c.categoryIndexError(w, err)
		return
	}

	categoryEntry, err := c.storage.GetCategory(ctx, int64(categoryID))

	if err != nil {
		c.categoryIndexError(w, err)
		return
	}

	// Get expenses for this category before update so we can return
	// If we have any error we want to still allow to display catgeory data
	expenses, err := c.storage.GetExpensesByCategory(ctx, categoryEntry.ID())

	if err != nil {
		c.categoryIndexError(w, err)
		return
	}

	enhancedCat := createEnhancedCategory(categoryEntry, expenses)

	updated := false
	patternChanged := false
	if (pattern != "" && categoryEntry.Pattern() != pattern) ||
		(name != "" && categoryEntry.Name() != name) {
		if pattern != "" && categoryEntry.Pattern() != pattern {
			_, err = regexp.Compile(pattern)

			if err != nil {
				enhancedCat.Errors = true
				enhancedCat.ErrorStrings = map[string]string{
					"pattern": fmt.Sprintf("invalid pattern %v", err),
				}

				c.templates.Render(w, "partials/categories/card.html", enhancedCat)
				return
			}
			patternChanged = true
		}

		err = c.storage.UpdateCategory(ctx, int64(categoryID), name, pattern)

		if err != nil {
			enhancedCat.Errors = true
			enhancedCat.ErrorStrings = map[string]string{
				"name": fmt.Sprintf("failed to updated category %v", err),
			}

			c.templates.Render(w, "partials/categories/card.html", enhancedCat)
			return
		}

		updated = true
		c.logger.Info("Category updated successfully", "id", categoryID)
	}

	//nolint:nestif // No need to extract this code to a function for now as is clear
	if updated {
		categories, categoryErr := c.storage.GetCategories(ctx)
		if err != nil {
			c.categoryIndexError(w, categoryErr)
			return
		}

		matcher := matcher.New(categories)
		c.matcher = matcher

		if patternChanged {
			allExpenses, expensesErr := c.storage.GetExpenses(ctx)

			if expensesErr != nil {
				c.categoryIndexError(w, expensesErr)
				return
			}

			toUpdated := []storage.Expense{}

			for _, ex := range allExpenses {
				id, _ := matcher.Match(ex.Description())
				if id != nil {
					if ex.CategoryID() != nil {
						// Update existing expense with a new category
						if *id != *ex.CategoryID() {
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
					} else {
						// Update existing expense without category with category
						expense := storage.NewExpense(ex.ID(), ex.Source(), ex.Description(), ex.Currency(), ex.Amount(), ex.Date(), ex.Type(), id)
						toUpdated = append(toUpdated, expense)
					}
				} else {
					if ex.CategoryID() != nil {
						// Changing a category pattern could render existing expenses to have category NULL
						expense := storage.NewExpense(ex.ID(), ex.Source(), ex.Description(), ex.Currency(), ex.Amount(), ex.Date(), ex.Type(), nil)
						toUpdated = append(toUpdated, expense)
					}
				}
			}

			if len(toUpdated) > 0 {
				updated, updateErr := c.storage.UpdateExpenses(ctx, toUpdated)
				if updateErr != nil {
					c.categoryIndexError(w, updateErr)
					return
				}

				c.logger.Info("Category's expenses updated successfully", "id", categoryID, "total", updated)

				if int(updated) != len(toUpdated) {
					c.logger.Warn("not all categories were updated")
				}
			}
		}
	}

	// Note: We need to refresh the category from storage since interfaces are immutable
	// The template will use the updated values from the storage

	if patternChanged {
		updateCategoryMatcherErr := c.updateCategoryMatcher()
		if updateCategoryMatcherErr != nil {
			c.categoryIndexError(w, updateCategoryMatcherErr)
			return
		}
	}

	if updated {
		c.resetCache()
	}

	updatedExpenses, err := c.storage.GetExpensesByCategory(ctx, categoryEntry.ID())

	if err != nil {
		c.categoryIndexError(w, err)
		return
	}

	updatedEnhancedCat := createEnhancedCategory(categoryEntry, updatedExpenses)

	c.templates.Render(w, "partials/categories/card.html", updatedEnhancedCat)
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
) {
	data := uncategorizedViewData{}
	data.CurrentPage = pageCategories
	var expenses []storage.Expense
	var err error

	if query != "" {
		expenses, err = c.storage.GetExpensesWithoutCategoryWithQuery(ctx, query)
	} else {
		expenses, err = c.storage.GetExpensesWithoutCategory(ctx)
	}

	if err != nil {
		data.Error = err.Error()
		c.templates.Render(w, "pages/categories/uncategorized.html", data)
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
	c.templates.Render(w, "pages/categories/uncategorized.html", data)
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
	data := viewBase{}

	err := r.ParseForm()
	if err != nil {
		c.logger.Error(fmt.Sprintf("error r.ParseForm() %s", err.Error()))

		data.Error = err.Error()
		c.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	expenseDescription := r.FormValue("description")
	categoryIDStr := r.FormValue("category_id")

	categoryID, err := strconv.Atoi(categoryIDStr)

	if err != nil {
		c.logger.Error(fmt.Sprintf("error strconv.Atoi with value %s. %s", categoryIDStr, err.Error()))

		data.Error = err.Error()
		c.templates.Render(w, "pages/categories/uncategorized.html", data)
	}

	cat, err := c.storage.GetCategory(ctx, int64(categoryID))

	if err != nil {
		c.logger.Error(fmt.Sprintf("error GetCategory %s", err.Error()))

		data.Error = err.Error()
		c.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	extendedRegex, err := extendRegex(cat.Pattern(), expenseDescription)

	if err != nil {
		c.logger.Error(fmt.Sprintf("error extendRegex %s", err.Error()))
		data.Error = err.Error()
		c.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	err = c.storage.UpdateCategory(ctx, cat.ID(), cat.Name(), extendedRegex)
	if err != nil {
		c.logger.Error(fmt.Sprintf("error UpdateCategory %s", err.Error()))
		data.Error = err.Error()
		c.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	c.logger.Info("Category updated successfully", "id", cat.ID(), "extended_regex", extendedRegex)

	expenses, err := c.storage.SearchExpensesByDescription(ctx, expenseDescription)

	if err != nil {
		c.logger.Error(fmt.Sprintf("error SearchExpensesByDescription %s", err.Error()))

		data.Error = err.Error()
		c.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	if len(expenses) > 0 {
		updatedExpenses := make([]storage.Expense, len(expenses))
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
		updated, updateErr := c.storage.UpdateExpenses(ctx, updatedExpenses)
		if updateErr != nil {
			data.Error = updateErr.Error()
			c.templates.Render(w, "pages/categories/uncategorized.html", data)
			return
		}

		c.logger.Info("Category's expenses updated successfully", "id", cat.ID(), "total", updated)

		if updated != int64(len(expenses)) {
			c.logger.Warn("not all expenses updated succesfully")
		}

		updateCategoryMatcherErr := c.updateCategoryMatcher()
		if updateCategoryMatcherErr != nil {
			data.Error = updateCategoryMatcherErr.Error()
			c.templates.Render(w, "pages/categories/uncategorized.html", data)
			return
		}

		c.resetCache()
	}

	c.uncategorizedHandler(ctx, w, "")
}

func (c *categoryHandler) resetcategoryHandler(ctx context.Context, w http.ResponseWriter) {
	_, err := c.storage.DeleteCategories(ctx)

	if err != nil {
		c.categoryIndexError(w, err)
		return
	}

	updateCategoryMatcherErr := c.updateCategoryMatcher()
	if updateCategoryMatcherErr != nil {
		c.categoryIndexError(w, updateCategoryMatcherErr)
		return
	}

	c.resetCache()

	c.categoriesHandler(ctx, w, nil, &banner{
		Icon:    "🔥",
		Message: "All categories deleted",
	})
}

func (c *categoryHandler) deletecategoryHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.categoriesHandler(ctx, w, fmt.Errorf("invalid ID. %s", err.Error()), nil)
		return
	}

	_, err = c.storage.DeleteCategory(ctx, id)
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
	data := createCategoryViewData{}
	data.CurrentPage = pageCategories
	template := "partials/categories/new_result.html"
	if create {
		template = "pages/categories/new.html"
	}

	err := r.ParseForm()
	if err != nil {
		c.logger.Error(fmt.Sprintf("error r.ParseForm() %s", err.Error()))

		data.Error = err.Error()
		c.templates.Render(w, template, data)
		return
	}

	name := r.FormValue("name")
	pattern := r.FormValue("pattern")

	data.Name = name
	data.Pattern = pattern

	if name == "" || pattern == "" {
		data.Error =
			"category must include name and a valid regex pattern. Ensure that you populate the name and pattern input"

		c.templates.Render(w, template, data)
		return
	}

	re, err := regexp.Compile(pattern)

	if err != nil {
		data.Error = err.Error()

		c.templates.Render(w, template, data)
		return
	}

	expenses, err := c.storage.GetExpensesWithoutCategory(ctx)

	if err != nil {
		data.Error = err.Error()

		c.templates.Render(w, template, data)
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
		categoryID, createErr := c.storage.CreateCategory(ctx, name, pattern)

		if createErr != nil {
			data.Error = createErr.Error()

			c.templates.Render(w, template, data)
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

		updated, updateErr := c.storage.UpdateExpenses(ctx, updatedExpenses)
		if updateErr != nil {
			data.Error = updateErr.Error()

			c.templates.Render(w, template, data)
			return
		}

		c.logger.Info("Category expenses updated", "total", updated)

		if int(updated) != total {
			c.logger.Warn("not all categories were updated")

			total = int(updated)
		}

		updateCategoryMatcherErr := c.updateCategoryMatcher()
		if updateCategoryMatcherErr != nil {
			data.Error = updateCategoryMatcherErr.Error()
			c.templates.Render(w, template, data)
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
		c.templates.Render(w, template, data)
		return
	}

	data.Total = total
	data.Results = toUpdated

	c.templates.Render(w, template, data)
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

func (c *categoryHandler) categoryIndexError(w http.ResponseWriter, err error) {
	data := struct {
		Error       string
		CurrentPage string
	}{
		Error:       err.Error(),
		CurrentPage: pageCategories,
	}
	c.templates.Render(w, "pages/categories/index.html", data)
}

func (c *categoryHandler) updateCategoryMatcher() error {
	categories, categoryErr := c.storage.GetCategories(context.Background())
	if categoryErr != nil {
		return categoryErr
	}

	matcher := matcher.New(categories)
	c.matcher = matcher
	return nil
}
