package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/GustavoCaso/expensetrace/internal/domain"
	"github.com/GustavoCaso/expensetrace/internal/service/category"
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
		base := viewBaseFromContext(r.Context())
		data := domain.CategoryViewData{
			ViewBase: base,
			Action:   newAction,
			Category: domain.EmptyCategory(),
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
		data := domain.ViewBase{}
		r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)
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

func (c *categoryHandler) categoriesHandler(
	ctx context.Context,
	w http.ResponseWriter,
	outerErr error,
	banner *domain.Banner,
) {
	userID := userIDFromContext(ctx)
	base := viewBaseFromContext(ctx)
	data := domain.CategoriesViewData{
		ViewBase: base,
	}

	defer func() {
		c.templates.Render(w, "pages/categories/index.html", data)
	}()

	enhancedCategories, categorizedCount, uncategorizedCount, err := c.categoryService.EnhancedList(ctx, userID)
	if err != nil {
		data.Error = err.Error()
		return
	}

	data.Categories = enhancedCategories
	data.CategorizedCount = categorizedCount
	data.UncategorizedCount = uncategorizedCount

	if outerErr != nil {
		data.Error = outerErr.Error()
	}

	if banner != nil {
		data.Banner = *banner
	}
}

func (c *categoryHandler) categoryHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	base := viewBaseFromContext(ctx)
	data := domain.CategoryViewData{
		ViewBase: base,
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

	categoryEntry, err := c.categoryService.Get(ctx, userID, id)
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
	base := viewBaseFromContext(ctx)
	data := domain.CategoryViewData{
		ViewBase: base,
		Action:   editAction,
	}
	data.Category = domain.EmptyCategory()

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

	// Parse form manually for partial updates
	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)
	if formErr := r.ParseForm(); formErr != nil {
		c.logger.Error("Failed to parse form", "error", formErr)
		data.Error = fmt.Errorf("invalid form data: %w", formErr).Error()
		return
	}

	name := r.FormValue("name")
	pattern := r.FormValue("pattern")
	budgetStr := r.FormValue("monthly_budget")

	updatedCategory, changed, patternDidChange, updateErr := c.categoryService.Update(
		ctx,
		userID,
		categoryIDInt64,
		name,
		pattern,
		budgetStr,
	)
	if updateErr != nil {
		c.logger.Error("Failed to update category", "error", updateErr)
		err = updateErr
		return
	}

	data.Category = updatedCategory

	if !changed {
		data.Banner = domain.Banner{
			Message: "Nothing to update",
		}
		return
	}

	c.logger.Info("Category updated successfully", "id", categoryID)

	data.Banner = domain.Banner{
		Icon:    "✅",
		Message: "Category Updated",
	}

	if !patternDidChange {
		return
	}

	if matcherErr := c.updateCategoryMatcher(ctx, userID); matcherErr != nil {
		err = matcherErr
		return
	}
}

func (c *categoryHandler) uncategorizedHandler(
	ctx context.Context,
	w http.ResponseWriter,
	query string,
	banner *domain.Banner,
) {
	userID := userIDFromContext(ctx)
	base := viewBaseFromContext(ctx)
	data := domain.UncategorizedViewData{
		ViewBase: base,
	}

	defer func() {
		c.templates.Render(w, "pages/categories/uncategorized.html", data)
	}()

	grouped, keys, totalExpenses, totalAmount, err := c.categoryService.GetUncategorized(ctx, userID, query)
	if err != nil {
		data.Error = err.Error()
		return
	}

	data.Keys = keys
	data.UncategorizeInfo = grouped
	data.Categories = c.matcher.Categories()
	data.TotalExpenses = totalExpenses
	data.TotalAmount = totalAmount

	if banner != nil {
		data.Banner = *banner
	}
}

func (c *categoryHandler) updateUncategorizedHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(ctx)
	data := viewBaseFromContext(ctx)

	defer func() {
		if data.Error != "" {
			c.templates.Render(w, "pages/categories/uncategorized.html", data)
		}
	}()

	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)
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

	if err = c.categoryService.UpdateCategoryPattern(ctx, userID, int64(categoryID), expenseDescription); err != nil {
		data.Error = err.Error()
		return
	}

	updateCategoryMatcherErr := c.updateCategoryMatcher(ctx, userID)
	if updateCategoryMatcherErr != nil {
		data.Error = updateCategoryMatcherErr.Error()
		return
	}

	c.uncategorizedHandler(ctx, w, "", &domain.Banner{
		Icon:    "✅",
		Message: "Expenses succesfully categorized",
	})
}

func (c *categoryHandler) resetCategoryHandler(ctx context.Context, w http.ResponseWriter) {
	userID := userIDFromContext(ctx)
	err := c.categoryService.Reset(ctx, userID)

	if err != nil {
		c.categoryIndexError(ctx, w, err)
		return
	}

	updateCategoryMatcherErr := c.updateCategoryMatcher(ctx, userID)
	if updateCategoryMatcherErr != nil {
		c.categoryIndexError(ctx, w, updateCategoryMatcherErr)
		return
	}

	c.categoriesHandler(ctx, w, nil, &domain.Banner{
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

	err = c.categoryService.Delete(ctx, userID, id)
	if err != nil {
		c.logger.Error("Failed to delete category", "error", err, "id", id)

		c.categoriesHandler(ctx, w, fmt.Errorf("error deleting the category. %s", err.Error()), nil)
		return
	}

	c.logger.Info("Category deleted successfully", "id", id)

	c.categoriesHandler(ctx, w, nil, &domain.Banner{
		Icon:    "✅",
		Message: "Category  deleted",
	})
}

func (c *categoryHandler) createCategoryHandler(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	userID := userIDFromContext(ctx)
	data := domain.CreateCategoryViewData{}
	data.LoggedIn = true
	data.CurrentPage = pageCategories
	data.Action = newAction

	defer func() {
		c.templates.Render(w, "pages/categories/new.html", data)
	}()

	// Initialize with empty category
	data.Category = domain.EmptyCategory()

	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)
	// Parse and validate form using helper
	formData, err := parseCategoryForm(r)
	if err != nil {
		c.logger.Error("Failed to parse category form", "error", err)
		data.Error = err.Error()
		return
	}

	// Store parsed values in category for re-rendering
	data.Category = domain.NewCategory(0, formData.Name, formData.Pattern, formData.MonthlyBudget)

	_, matched, createErr := c.categoryService.Create(ctx, userID, *formData)
	if createErr != nil {
		c.logger.Error("Failed to create category", "error", createErr)
		data.Error = createErr.Error()
		return
	}

	updateCategoryMatcherErr := c.updateCategoryMatcher(ctx, userID)
	if updateCategoryMatcherErr != nil {
		c.logger.Error("Failed to update category matcher", "error", updateCategoryMatcherErr)
		data.Error = updateCategoryMatcherErr.Error()
		return
	}

	data.Banner = domain.Banner{
		Icon: "✅",
		Message: fmt.Sprintf(
			"Category %s was created successfully and %d transactions were categorized!",
			formData.Name,
			len(matched),
		),
	}

	data.Total = len(matched)
	data.Results = matched
}

func (c *categoryHandler) testCategoryHandler(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) {
	userID := userIDFromContext(ctx)
	data := domain.CreateCategoryViewData{}
	data.LoggedIn = true
	data.CurrentPage = pageCategories

	defer func() {
		c.templates.Render(w, "partials/categories/test_category.html", data)
	}()

	r.Body = http.MaxBytesReader(w, r.Body, maxFormSize)
	// Parse and validate form using helper
	formData, err := parseCategoryForm(r)
	if err != nil {
		c.logger.Error("Failed to parse category form", "error", err)
		data.Error = err.Error()
		return
	}

	// Store parsed values in category for re-rendering
	data.Category = domain.NewCategory(0, formData.Name, formData.Pattern, formData.MonthlyBudget)

	matched, testErr := c.categoryService.Test(ctx, userID, formData.Pattern)
	if testErr != nil {
		c.logger.Error("Failed to test category pattern", "error", testErr)
		data.Error = testErr.Error()
		return
	}

	data.Total = len(matched)
	data.Results = matched
}

func (c *categoryHandler) categoryIndexError(ctx context.Context, w http.ResponseWriter, err error) {
	data := viewBaseFromContext(ctx)
	data.Error = err.Error()
	c.templates.Render(w, "pages/categories/index.html", data)
}

// parseCategoryForm parses and validates category form fields from the request.
// Returns parsed data or an error with a user-friendly message.
func parseCategoryForm(r *http.Request) (*domain.CategoryFormData, error) {
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
	monthlyBudget, budgetErr := category.ValidateBudget(budgetStr)
	if budgetErr != nil {
		return nil, budgetErr
	}

	return &domain.CategoryFormData{
		Name:          name,
		Pattern:       pattern,
		MonthlyBudget: monthlyBudget,
	}, nil
}
