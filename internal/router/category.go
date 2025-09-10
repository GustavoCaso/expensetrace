package router

import (
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

func (router *router) categoriesHandler(w http.ResponseWriter) {
	categories, err := router.storage.GetCategories()

	if err != nil {
		categoryIndexError(router, w, fmt.Errorf("error fetch categories: %s", err.Error()))
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
	uncategorizedInfos, err := router.storage.GetExpensesWithoutCategory()
	if err != nil {
		categoryIndexError(router, w, err)
		return
	}
	uncategorizedCount := len(uncategorizedInfos)

	// Get total categorized count
	totalCategorized := 0

	// Enhance categories with additional data
	enhancedCategories := make([]enhancedCategory, len(categoriesWithoutExclude))

	for i, cat := range categoriesWithoutExclude {
		// Get expenses for this category
		expenses, expensesErr := router.storage.GetExpensesByCategory(cat.ID())

		if expensesErr != nil {
			categoryIndexError(router, w, expensesErr)
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

	router.templates.Render(w, "pages/categories/index.html", data)
}

func (router *router) updateCategoryHandler(
	id, name, pattern string,
	w http.ResponseWriter,
) {
	categoryID, err := strconv.Atoi(id)

	if err != nil {
		categoryIndexError(router, w, err)
		return
	}

	categoryEntry, err := router.storage.GetCategory(int64(categoryID))

	if err != nil {
		categoryIndexError(router, w, err)
		return
	}

	// Get expenses for this category
	expenses, err := router.storage.GetExpensesByCategory(categoryEntry.ID())

	if err != nil {
		categoryIndexError(router, w, err)
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

				router.templates.Render(w, "partials/categories/card.html", enhancedCat)
				return
			}
			patternChanged = true
		}

		err = router.storage.UpdateCategory(int64(categoryID), name, pattern)

		if err != nil {
			enhancedCat.Errors = true
			enhancedCat.ErrorStrings = map[string]string{
				"name": fmt.Sprintf("failed to updated category %v", err),
			}

			router.templates.Render(w, "partials/categories/card.html", enhancedCat)
			return
		}

		updated = true
		router.logger.Info("Category updated successfully", "id", categoryID)
	}

	//nolint:nestif // No need to extract this code to a function for now as is clear
	if updated {
		categories, categoryErr := router.storage.GetCategories()
		if err != nil {
			categoryIndexError(router, w, categoryErr)
			return
		}

		matcher := matcher.New(categories)
		router.matcher = matcher

		if patternChanged {
			allExpenses, expensesErr := router.storage.GetExpenses()

			if expensesErr != nil {
				categoryIndexError(router, w, expensesErr)
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
				updated, updateErr := router.storage.UpdateExpenses(toUpdated)
				if updateErr != nil {
					categoryIndexError(router, w, updateErr)
					return
				}

				router.logger.Info("Category's expenses updated successfully", "id", categoryID, "total", updated)

				if int(updated) != len(toUpdated) {
					router.logger.Warn("not all categories were updated")
				}
			}
		}
	}

	// Note: We need to refresh the category from storage since interfaces are immutable
	// The template will use the updated values from the storage

	if patternChanged {
		updateCategoryMatcherErr := router.updateCategoryMatcher()
		if updateCategoryMatcherErr != nil {
			categoryIndexError(router, w, updateCategoryMatcherErr)
			return
		}
	}

	if updated {
		router.resetCache()
	}

	router.templates.Render(w, "partials/categories/card.html", enhancedCat)
}

type uncategorizedInfo struct {
	Count    int
	Expenses []struct {
		Date   time.Time
		Amount int64
		Source string
	}
	Total int64
	Slug  string
}

type uncategorizedViewData struct {
	viewBase
	Keys             []string
	UncategorizeInfo map[string]uncategorizedInfo
	Categories       []storage.Category
	TotalExpenses    int
	TotalAmount      int64
}

func (router *router) uncategorizedHandler(w http.ResponseWriter) {
	data := uncategorizedViewData{}
	expenses, err := router.storage.GetExpensesWithoutCategory()
	if err != nil {
		data.Error = err.Error()
		router.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	uncategorizeInfo := map[string]uncategorizedInfo{}
	totalExpenses := 0
	var totalAmount int64

	for _, ex := range expenses {
		if r, ok := uncategorizeInfo[ex.Description()]; ok {
			r.Count++
			r.Expenses = append(r.Expenses, struct {
				Date   time.Time
				Amount int64
				Source string
			}{
				Date:   ex.Date(),
				Amount: ex.Amount(),
				Source: ex.Source(),
			})
			r.Total += ex.Amount()
			uncategorizeInfo[ex.Description()] = r
		} else {
			uncategorizeInfo[ex.Description()] = uncategorizedInfo{
				Count: 1,
				Total: ex.Amount(),
				Expenses: []struct {
					Date   time.Time
					Amount int64
					Source string
				}{
					{
						Date:   ex.Date(),
						Amount: ex.Amount(),
						Source: ex.Source(),
					},
				},
				Slug: slugify(ex.Description()),
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
			return report.Expenses[i].Date.After(report.Expenses[j].Date)
		})
	}

	data.Keys = keys
	data.UncategorizeInfo = uncategorizeInfo
	data.Categories = router.matcher.Categories()
	data.TotalExpenses = totalExpenses
	data.TotalAmount = totalAmount
	router.templates.Render(w, "pages/categories/uncategorized.html", data)
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

func (router *router) updateUncategorizedHandler(w http.ResponseWriter, r *http.Request) {
	data := viewBase{}

	err := r.ParseForm()
	if err != nil {
		router.logger.Error(fmt.Sprintf("error r.ParseForm() %s", err.Error()))

		data.Error = err.Error()
		router.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	expenseDescription := r.FormValue("description")
	categoryIDStr := r.FormValue("categoryID")
	categoryID, err := strconv.Atoi(categoryIDStr)

	if err != nil {
		router.logger.Error(fmt.Sprintf("error strconv.Atoi with value %s. %s", categoryIDStr, err.Error()))

		data.Error = err.Error()
		router.templates.Render(w, "pages/categories/uncategorized.html", data)
	}

	cat, err := router.storage.GetCategory(int64(categoryID))

	if err != nil {
		router.logger.Error(fmt.Sprintf("error GetCategory %s", err.Error()))

		data.Error = err.Error()
		router.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	extendedRegex, err := extendRegex(cat.Pattern(), expenseDescription)

	if err != nil {
		router.logger.Error(fmt.Sprintf("error extendRegex %s", err.Error()))
		data.Error = err.Error()
		router.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	err = router.storage.UpdateCategory(cat.ID(), cat.Name(), extendedRegex)
	if err != nil {
		router.logger.Error(fmt.Sprintf("error UpdateCategory %s", err.Error()))
		data.Error = err.Error()
		router.templates.Render(w, "pages/categories/uncategorized.html", data)
		return
	}

	router.logger.Info("Category updated successfully", "id", cat.ID(), "extended_regex", extendedRegex)

	expenses, err := router.storage.SearchExpensesByDescription(expenseDescription)

	if err != nil {
		router.logger.Error(fmt.Sprintf("error SearchExpensesByDescription %s", err.Error()))

		data.Error = err.Error()
		router.templates.Render(w, "pages/categories/uncategorized.html", data)
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
		updated, updateErr := router.storage.UpdateExpenses(updatedExpenses)
		if updateErr != nil {
			data.Error = updateErr.Error()
			router.templates.Render(w, "pages/categories/uncategorized.html", data)
			return
		}

		router.logger.Info("Category's expenses updated successfully", "id", cat.ID(), "total", updated)

		if updated != int64(len(expenses)) {
			router.logger.Warn("not all expenses updated succesfully")
		}

		updateCategoryMatcherErr := router.updateCategoryMatcher()
		if updateCategoryMatcherErr != nil {
			data.Error = updateCategoryMatcherErr.Error()
			router.templates.Render(w, "pages/categories/uncategorized.html", data)
			return
		}

		router.resetCache()
	}

	router.uncategorizedHandler(w)
}

func (router *router) resetCategoryHandler(w http.ResponseWriter) {
	_, err := router.storage.DeleteCategories()

	if err != nil {
		categoryIndexError(router, w, err)
		return
	}

	updateCategoryMatcherErr := router.updateCategoryMatcher()
	if updateCategoryMatcherErr != nil {
		categoryIndexError(router, w, updateCategoryMatcherErr)
		return
	}

	router.resetCache()

	router.categoriesHandler(w)
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
	Create  bool
}

func (router *router) createCategoryHandler(create bool, w http.ResponseWriter, r *http.Request) {
	data := createCategoryViewData{}
	err := r.ParseForm()
	if err != nil {
		router.logger.Error(fmt.Sprintf("error r.ParseForm() %s", err.Error()))

		data.Error = err.Error()
		router.templates.Render(w, "partials/categories/new_result.html", data)
		return
	}

	name := r.FormValue("name")
	pattern := r.FormValue("pattern")

	data.Name = name
	data.Pattern = pattern

	if name == "" || pattern == "" {
		data.Error =
			"category must include name and a valid regex pattern. Ensure that you populate the name and pattern input"

		router.templates.Render(w, "partials/categories/new_result.html", data)
		return
	}

	re, err := regexp.Compile(pattern)

	if err != nil {
		data.Error = err.Error()

		router.templates.Render(w, "partials/categories/new_result.html", data)
		return
	}

	expenses, err := router.storage.GetExpensesWithoutCategory()

	if err != nil {
		data.Error = err.Error()

		router.templates.Render(w, "partials/categories/new_result.html", data)
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
		categoryID, createErr := router.storage.CreateCategory(name, pattern)

		if createErr != nil {
			data.Error = createErr.Error()

			router.templates.Render(w, "partials/categories/new_result.html", data)
			return
		}

		router.logger.Info("Category created", "name", name, "pattern", pattern)

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

		updated, updateErr := router.storage.UpdateExpenses(updatedExpenses)
		if updateErr != nil {
			data.Error = updateErr.Error()

			router.templates.Render(w, "partials/categories/new_result.html", data)
			return
		}

		router.logger.Info("Category expenses updated", "total", updated)

		if int(updated) != total {
			router.logger.Warn("not all categories were updated")

			total = int(updated)
		}

		updateCategoryMatcherErr := router.updateCategoryMatcher()
		if updateCategoryMatcherErr != nil {
			data.Error = updateCategoryMatcherErr.Error()
			router.templates.Render(w, "partials/categories/new_result.html", data)
			return
		}

		router.resetCache()
	}

	data.Total = total
	data.Results = toUpdated
	data.Create = create

	router.templates.Render(w, "partials/categories/new_result.html", data)
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

func categoryIndexError(router *router, w http.ResponseWriter, err error) {
	data := struct {
		Error string
	}{
		Error: err.Error(),
	}
	router.templates.Render(w, "pages/categories/index.html", data)
}

func (router *router) updateCategoryMatcher() error {
	categories, categoryErr := router.storage.GetCategories()
	if categoryErr != nil {
		return categoryErr
	}

	matcher := matcher.New(categories)
	router.matcher = matcher
	return nil
}
