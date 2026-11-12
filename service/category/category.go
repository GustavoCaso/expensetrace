package category

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/logger"
	"github.com/GustavoCaso/expensetrace/matcher"
	"github.com/GustavoCaso/expensetrace/service/report"
	"github.com/GustavoCaso/expensetrace/storage"
)

type Service struct {
	storage storage.Storage
	logger  *logger.Logger
}

func New(storage storage.Storage, logger *logger.Logger) *Service {
	return &Service{
		storage: storage,
		logger:  logger,
	}
}

// List returns all of the user's categories, excluding the special exclude
// category.
func (c *Service) List(ctx context.Context, userID int64) ([]domain.Category, error) {
	return c.storage.GetCategories(ctx, userID)
}

// Get returns a single category by ID.
func (c *Service) Get(ctx context.Context, userID, id int64) (domain.Category, error) {
	return c.storage.GetCategory(ctx, userID, id)
}

// EnhancedList returns all of the user's categories (excluding the exclude
// category) enhanced with spending statistics, along with the total number
// of categorized and uncategorized expenses.
func (c *Service) EnhancedList(
	ctx context.Context,
	userID int64,
) ([]domain.EnhancedCategory, int, int, error) {
	cats, err := c.List(ctx, userID)
	if err != nil {
		return nil, 0, 0, err
	}

	uncategorizedInfos, err := c.storage.GetExpensesWithoutCategory(ctx, userID)
	if err != nil {
		return nil, 0, 0, err
	}
	uncategorizedCount := len(uncategorizedInfos)

	enhancedCategories := make([]domain.EnhancedCategory, len(cats))
	now := time.Now()

	categorizedCount := 0
	for i, cat := range cats {
		expenses, expensesErr := c.storage.GetExpensesByCategory(ctx, userID, cat.ID())
		if expensesErr != nil {
			return nil, 0, 0, expensesErr
		}

		categorizedCount += len(expenses)
		enhancedCategories[i] = createEnhancedCategory(cat, expenses, now)
	}

	return enhancedCategories, categorizedCount, uncategorizedCount, nil
}

// Delete deletes a category.
func (c *Service) Delete(ctx context.Context, userID, id int64) error {
	_, err := c.storage.DeleteCategory(ctx, userID, id)
	return err
}

// Reset deletes all of the user's categories.
func (c *Service) Reset(ctx context.Context, userID int64) error {
	_, err := c.storage.DeleteCategories(ctx, userID)
	return err
}

// ValidateBudget parses and validates a monthly budget string (expressed as
// a decimal amount, e.g. "100.50"), returning the equivalent value in cents.
func ValidateBudget(budgetStr string) (int64, error) {
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

// createEnhancedCategory computes the category's statistics; "this month" is
// the calendar month containing now.
func createEnhancedCategory(
	category domain.Category,
	expenses []domain.Expense,
	now time.Time,
) domain.EnhancedCategory {
	var totalAmount, monthAmount int64
	var lastTransaction time.Time
	spendingCount := 0
	incomeCount := 0

	for _, exp := range expenses {
		totalAmount += exp.Amount()

		if exp.Date().Year() == now.Year() && exp.Date().Month() == now.Month() {
			monthAmount += exp.Amount()
		}

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

	budget := domain.BudgetInfo{Status: domain.BudgetStatusNoBudget}
	if category.Type() == domain.ChargeCategory && category.MonthlyBudget() > 0 {
		budget = report.CalculateBudgetInfo(category.MonthlyBudget(), monthAmount)
	}

	return domain.EnhancedCategory{
		Category:        category,
		AvgAmount:       avgAmount,
		LastTransaction: lastTransactionStr,
		Total:           len(expenses),
		TotalAmount:     totalAmount,
		SpendingCount:   spendingCount,
		IncomeCount:     incomeCount,
		MonthAmount:     monthAmount,
		Budget:          budget,
	}
}

func (c *Service) UpdateCategoryPattern(
	ctx context.Context,
	userID, categoryID int64,
	description string,
) error {
	cat, err := c.storage.GetCategory(ctx, userID, categoryID)

	if err != nil {
		c.logger.Error(fmt.Sprintf("error GetCategory %s", err.Error()))

		return err
	}

	extendedRegex, err := extendRegex(cat.Pattern(), description)

	if err != nil {
		c.logger.Error(fmt.Sprintf("error extendRegex %s", err.Error()))
		return err
	}

	err = c.storage.UpdateCategory(ctx, userID, cat.ID(), cat.Name(), extendedRegex, cat.MonthlyBudget(), cat.Type())
	if err != nil {
		c.logger.Error(fmt.Sprintf("error UpdateCategory %s", err.Error()))
		return err
	}

	c.logger.Info("Category updated successfully", "id", cat.ID(), "extended_regex", extendedRegex)

	expenses, err := c.storage.SearchExpensesByDescription(ctx, userID, description)

	if err != nil {
		c.logger.Error(fmt.Sprintf("error SearchExpensesByDescription %s", err.Error()))

		return err
	}

	// Only expenses the category can hold are moved into it.
	expenses = slices.DeleteFunc(expenses, func(ex domain.Expense) bool {
		return !cat.Type().Accepts(ex.Type())
	})

	updatedExpenses := make([]domain.Expense, len(expenses))

	if len(expenses) > 0 {
		for i, ex := range expenses {
			expense := domain.NewExpense(
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
			return updateErr
		}

		c.logger.Info("Category's expenses updated successfully", "id", cat.ID(), "total", updated)

		if updated != int64(len(expenses)) {
			c.logger.Warn("not all expenses updated succesfully")
		}
	}

	return nil
}

// Create creates a new category, and categorizes any currently uncategorized
// expenses whose description matches the category's pattern.
func (c *Service) Create(
	ctx context.Context,
	userID int64,
	form domain.CategoryFormData,
) (int64, []domain.Expense, error) {
	expenses, err := c.storage.GetExpensesWithoutCategory(ctx, userID)
	if err != nil {
		c.logger.Error("Failed to get expenses without category", "error", err)
		return 0, nil, err
	}

	re, err := regexp.Compile(form.Pattern)
	if err != nil {
		return 0, nil, err
	}

	toUpdated := []domain.Expense{}

	for _, ex := range expenses {
		if form.Type.Accepts(ex.Type()) && re.MatchString(ex.Description()) {
			toUpdated = append(toUpdated, ex)
		}
	}

	monthlyBudget := form.MonthlyBudget
	if form.Type == domain.IncomeCategory {
		// Budgets only make sense for spending.
		monthlyBudget = 0
	}

	categoryID, err := c.storage.CreateCategory(ctx, userID, form.Name, form.Pattern, monthlyBudget, form.Type)
	if err != nil {
		c.logger.Error("Failed to create category", "error", err)
		return 0, nil, err
	}

	c.logger.Info("Category created", "name", form.Name, "pattern", form.Pattern, "type", form.Type.String())

	if len(toUpdated) > 0 {
		updatedExpenses := make([]domain.Expense, len(toUpdated))

		for i, ex := range toUpdated {
			expense := domain.NewExpense(
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
			return categoryID, toUpdated, updateErr
		}

		c.logger.Info("Category expenses updated", "total", updated)

		if int(updated) != len(toUpdated) {
			c.logger.Warn("not all categories were updated")
		}
	}

	return categoryID, toUpdated, nil
}

// Update updates a category's name, pattern, monthly budget and/or type, given
// the raw form values (empty name/pattern/budgetStr/typeStr means "keep
// existing"). Changing the type fails while the category holds expenses the
// new type cannot accept. It
// resolves defaults and computes whether anything actually changed
// internally, so callers do not need to pre-fetch the category or duplicate
// the diff logic. If nothing changed, no storage write is performed and
// changed=false is returned. If the pattern or type changes, it re-sweeps the
// user's uncategorized and current-category expenses, recategorizing them
// based on the new pattern set. It returns the updated category, whether
// anything changed, and whether matching specifically changed (so the caller
// can decide whether to refresh its own matcher).
func (c *Service) Update(
	ctx context.Context,
	userID, categoryID int64,
	name, pattern, budgetStr, typeStr string,
) (domain.Category, bool, bool, error) {
	existingCategory, err := c.storage.GetCategory(ctx, userID, categoryID)
	if err != nil {
		c.logger.Error(fmt.Sprintf("error GetCategory %s", err.Error()))
		return domain.EmptyCategory(), false, false, err
	}

	if name == "" {
		name = existingCategory.Name()
	}

	if pattern == "" {
		pattern = existingCategory.Pattern()
	}

	monthlyBudget := existingCategory.MonthlyBudget()
	if budgetStr != "" {
		monthlyBudget, err = ValidateBudget(budgetStr)
		if err != nil {
			c.logger.Error(fmt.Sprintf("error ValidateBudget %s", err.Error()))
			return domain.EmptyCategory(), false, false, err
		}
	}

	if _, compileErr := regexp.Compile(pattern); compileErr != nil {
		c.logger.Error(fmt.Sprintf("error invalid pattern %s", compileErr.Error()))
		return domain.EmptyCategory(), false, false, compileErr
	}

	categoryType := existingCategory.Type()
	// The exclude category always accepts every expense type.
	if typeStr != "" && categoryType != domain.AnyCategory {
		categoryType, err = domain.ParseCategoryType(typeStr)
		if err != nil {
			return domain.EmptyCategory(), false, false, err
		}
	}

	if categoryType == domain.IncomeCategory {
		monthlyBudget = 0
	}

	nameChanged := existingCategory.Name() != name
	patternChanged := existingCategory.Pattern() != pattern
	budgetChanged := monthlyBudget != existingCategory.MonthlyBudget()
	typeChanged := categoryType != existingCategory.Type()

	if !nameChanged && !patternChanged && !budgetChanged && !typeChanged {
		return existingCategory, false, false, nil
	}

	if typeChanged {
		if typeErr := c.checkTypeChange(ctx, userID, categoryID, categoryType); typeErr != nil {
			return domain.EmptyCategory(), false, false, typeErr
		}
	}

	matchingChanged := patternChanged || typeChanged

	err = c.storage.UpdateCategory(ctx, userID, categoryID, name, pattern, monthlyBudget, categoryType)
	if err != nil {
		c.logger.Error(fmt.Sprintf("error UpdateCategory %s", err.Error()))
		return domain.EmptyCategory(), false, false, err
	}

	c.logger.Info("Category updated successfully", "id", categoryID)

	updatedCategory, err := c.storage.GetCategory(ctx, userID, categoryID)
	if err != nil {
		c.logger.Error(fmt.Sprintf("error GetCategory %s", err.Error()))
		return domain.EmptyCategory(), true, matchingChanged, err
	}

	if !matchingChanged {
		return updatedCategory, true, false, nil
	}

	return updatedCategory, true, true, c.recategorize(ctx, userID, categoryID)
}

// recategorize re-matches the user's uncategorized expenses and those in
// categoryID against the current category set, after categoryID's matching
// rules changed.
func (c *Service) recategorize(ctx context.Context, userID, categoryID int64) error {
	categories, err := c.storage.GetCategories(ctx, userID)
	if err != nil {
		c.logger.Error(fmt.Sprintf("error GetCategories %s", err.Error()))
		return err
	}

	m := matcher.New(categories)

	uncategorizedExpenses, err := c.storage.GetExpensesWithoutCategory(ctx, userID)
	if err != nil {
		c.logger.Error(fmt.Sprintf("error GetExpensesWithoutCategory %s", err.Error()))
		return err
	}

	currentCategoryExpenses, err := c.storage.GetExpensesByCategory(ctx, userID, categoryID)
	if err != nil {
		c.logger.Error(fmt.Sprintf("error GetExpensesByCategory %s", err.Error()))
		return err
	}

	// Combine both sets of expenses to process
	expensesToProcess := make([]domain.Expense, 0, len(currentCategoryExpenses)+len(uncategorizedExpenses))
	expensesToProcess = append(expensesToProcess, currentCategoryExpenses...)
	expensesToProcess = append(expensesToProcess, uncategorizedExpenses...)
	toUpdated := []domain.Expense{}

	for _, ex := range expensesToProcess {
		id, _ := m.Match(ex.Description(), ex.Type())

		// 1. match && expense does not have a category OR the existing category is different
		// 2. no match && expense is part of the category we are updating
		if (id != nil && (ex.CategoryID() == nil || *ex.CategoryID() != *id)) ||
			(id == nil && expenseBelongsToCategoryWeAreUpdating(ex, categoryID)) {
			expense := domain.NewExpense(
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
		updatedCount, updatedErr := c.storage.UpdateExpenses(ctx, userID, toUpdated)
		if updatedErr != nil {
			c.logger.Error(fmt.Sprintf("error UpdateExpenses %s", updatedErr.Error()))
			return updatedErr
		}

		c.logger.Info("Category's expenses updated successfully", "id", categoryID, "total", updatedCount)

		if int(updatedCount) != len(toUpdated) {
			c.logger.Warn("not all categories were updated")
		}
	}

	return nil
}

// checkTypeChange fails if the category holds expenses that a category of
// type newType cannot accept.
func (c *Service) checkTypeChange(
	ctx context.Context,
	userID, categoryID int64,
	newType domain.CategoryType,
) error {
	expenses, err := c.storage.GetExpensesByCategory(ctx, userID, categoryID)
	if err != nil {
		return err
	}

	mismatched := 0
	for _, ex := range expenses {
		if !newType.Accepts(ex.Type()) {
			mismatched++
		}
	}

	if mismatched > 0 {
		return fmt.Errorf(
			"cannot change category type to %s: it holds %d expenses of the other type",
			newType, mismatched,
		)
	}

	return nil
}

func expenseBelongsToCategoryWeAreUpdating(ex domain.Expense, categoryID int64) bool {
	return ex.CategoryID() != nil && *ex.CategoryID() == categoryID
}

// GetUncategorized fetches uncategorized expenses (optionally filtered by
// query) and groups them by description and type, so a description seen as
// both a charge and income yields two groups. Groups are ordered by count
// (descending), then description, then type; each group's expenses are newest
// first. It also returns the overall totals.
func (c *Service) GetUncategorized(
	ctx context.Context,
	userID int64,
	query string,
) ([]domain.UncategorizedInfo, int, int64, error) {
	var expenses []domain.Expense
	var err error

	if query != "" {
		expenses, err = c.storage.GetExpensesWithoutCategoryWithQuery(ctx, userID, query)
	} else {
		expenses, err = c.storage.GetExpensesWithoutCategory(ctx, userID)
	}

	if err != nil {
		return nil, 0, 0, err
	}

	type groupKey struct {
		description string
		expenseType domain.ExpenseType
	}

	index := map[groupKey]int{}
	groups := []domain.UncategorizedInfo{}
	var totalAmount int64

	for _, ex := range expenses {
		key := groupKey{ex.Description(), ex.Type()}
		i, ok := index[key]
		if !ok {
			i = len(groups)
			index[key] = i
			groups = append(groups, domain.UncategorizedInfo{
				Description: ex.Description(),
				Type:        ex.Type(),
			})
		}

		groups[i].Count++
		groups[i].Total += ex.Amount()
		groups[i].Expenses = append(groups[i].Expenses, ex)
		totalAmount += ex.Amount()
	}

	slices.SortStableFunc(groups, func(a, b domain.UncategorizedInfo) int {
		if c := cmp.Compare(b.Count, a.Count); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Description, b.Description); c != 0 {
			return c
		}
		return cmp.Compare(a.Type, b.Type)
	})

	for _, group := range groups {
		slices.SortStableFunc(group.Expenses, func(a, b domain.Expense) int {
			return b.Date().Compare(a.Date())
		})
	}

	return groups, len(expenses), totalAmount, nil
}

// Test returns the uncategorized expenses of a type categoryType accepts that
// would match the given pattern, without writing anything to storage.
func (c *Service) Test(
	ctx context.Context,
	userID int64,
	pattern string,
	categoryType domain.CategoryType,
) ([]domain.Expense, error) {
	expenses, err := c.storage.GetExpensesWithoutCategory(ctx, userID)
	if err != nil {
		c.logger.Error("Failed to get expenses without category", "error", err)
		return nil, err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	matched := []domain.Expense{}

	for _, ex := range expenses {
		if categoryType.Accepts(ex.Type()) && re.MatchString(ex.Description()) {
			matched = append(matched, ex)
		}
	}

	return matched, nil
}

func extendRegex(pattern, description string) (string, error) {
	extendedPattern := fmt.Sprintf("%s|%s", pattern, regexp.QuoteMeta(description))
	re, err := regexp.Compile(extendedPattern)
	if err != nil {
		return "", err
	}
	return re.String(), nil
}
