package category

import (
	"context"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/testutil"
)

func TestServiceList_IncludesExcludeCategory(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	_, err := s.CreateCategory(
		context.Background(),
		user.ID(),
		"Entertainment",
		"cinema|movie",
		0,
		domain.ChargeCategory,
	)
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	svc := New(s, logger)

	categories, err := svc.List(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(categories) != 2 {
		t.Fatalf("Expected 2 category, got %d", len(categories))
	}

	foundExclude := false
	foundEntertaiment := false
	for _, c := range categories {
		if c.Name() == domain.ExcludeCategory {
			foundExclude = true
		}
		if c.Name() == "Entertainment" {
			foundEntertaiment = true
		}
	}

	if !foundExclude {
		t.Fatal("Expected exclude category to be included")
	}

	if !foundEntertaiment {
		t.Fatalf("Expected 'Entertainment' category to be included")
	}
}

func TestServiceCreate_MatchesExistingUncategorizedExpenses(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "cinema", "USD", -123456, time.Now(), domain.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)

	categoryID, matched, err := svc.Create(context.Background(), user.ID(), domain.CategoryFormData{
		Name:          "Entertainment",
		Pattern:       "cinema|movie|theater",
		MonthlyBudget: 10000,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if categoryID == 0 {
		t.Fatal("Expected non-zero category ID")
	}

	if len(matched) != 1 {
		t.Fatalf("Expected 1 matched expense, got %d", len(matched))
	}

	updatedExpenses, err := s.SearchExpensesByDescription(context.Background(), user.ID(), "cinema")
	if err != nil {
		t.Fatalf("Failed to search expenses: %v", err)
	}

	if len(updatedExpenses) != 1 {
		t.Fatalf("Expected 1 expense, got %d", len(updatedExpenses))
	}

	if updatedExpenses[0].CategoryID() == nil || *updatedExpenses[0].CategoryID() != categoryID {
		t.Fatalf("Expense category was not updated to %d", categoryID)
	}
}

func TestServiceUpdate_PatternChangeRecategorizesExpenses(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory(
		context.Background(),
		user.ID(),
		"Entertainment",
		"restaurant|bars|cinema",
		0,
		domain.ChargeCategory,
	)
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "cinema", "USD", -123456, time.Now(), domain.ChargeType, &categoryID),
		domain.NewExpense(0, "Test Source", "gym", "USD", -123, time.Now(), domain.ChargeType, nil),
	}

	_, err = s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)

	// Need a matcher reflecting the *new* pattern for recategorization.
	// The service Update method itself must recompute matches using regex,
	// consistent with router's updateCategoryMatcher + matcher.Match flow.
	updatedCategory, changed, patternChanged, err := svc.Update(
		context.Background(),
		user.ID(),
		categoryID,
		"",
		"restaurant|bars|cinema|gym",
		"",
		"",
	)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if !changed {
		t.Fatal("Expected changed to be true")
	}

	if !patternChanged {
		t.Fatal("Expected patternChanged to be true")
	}

	if updatedCategory.Pattern() != "restaurant|bars|cinema|gym" {
		t.Fatalf("Expected pattern to be updated, got %s", updatedCategory.Pattern())
	}

	allExpenses, err := s.GetExpenses(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	for _, ex := range allExpenses {
		if ex.Description() == "gym" {
			if ex.CategoryID() == nil || *ex.CategoryID() != categoryID {
				t.Fatalf("Expected gym expense to be categorized into %d, got %v", categoryID, ex.CategoryID())
			}
		}
	}
}

func TestServiceUpdate_NoopWhenNothingChanged(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory(
		context.Background(),
		user.ID(),
		"Entertainment",
		"restaurant|bars|cinema",
		0,
		domain.ChargeCategory,
	)
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	svc := New(s, logger)

	updatedCategory, changed, patternChanged, err := svc.Update(
		context.Background(),
		user.ID(),
		categoryID,
		"Entertainment",
		"restaurant|bars|cinema",
		"",
		"",
	)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if changed {
		t.Fatal("Expected changed to be false")
	}

	if patternChanged {
		t.Fatal("Expected patternChanged to be false")
	}

	if updatedCategory.Name() != "Entertainment" || updatedCategory.Pattern() != "restaurant|bars|cinema" {
		t.Fatalf("Expected category to remain unchanged, got name=%s pattern=%s",
			updatedCategory.Name(), updatedCategory.Pattern())
	}
}

func TestServiceUpdateCategoryPattern_ExtendsPatternAndRecategorizes(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	categoryID, err := s.CreateCategory(
		context.Background(),
		user.ID(),
		"Entertainment",
		"cinema|movie",
		0,
		domain.ChargeCategory,
	)
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "netflix", "USD", -1500, time.Now(), domain.ChargeType, nil),
		domain.NewExpense(0, "Test Source", "cinema", "USD", -1000, time.Now(), domain.ChargeType, &categoryID),
	}

	_, err = s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)

	err = svc.UpdateCategoryPattern(context.Background(), user.ID(), categoryID, "netflix")
	if err != nil {
		t.Fatalf("UpdateCategoryPattern returned error: %v", err)
	}

	updatedCategory, err := s.GetCategory(context.Background(), user.ID(), categoryID)
	if err != nil {
		t.Fatalf("Failed to get category: %v", err)
	}

	if updatedCategory.Pattern() != "cinema|movie|netflix" {
		t.Fatalf("Expected pattern to be extended, got %s", updatedCategory.Pattern())
	}

	allExpenses, err := s.GetExpenses(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	for _, ex := range allExpenses {
		if ex.Description() == "netflix" {
			if ex.CategoryID() == nil || *ex.CategoryID() != categoryID {
				t.Fatalf("Expected netflix expense to be categorized into %d, got %v", categoryID, ex.CategoryID())
			}
		}
	}
}

func TestServiceUpdateCategoryPattern_ReturnsErrorWhenCategoryNotFound(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	svc := New(s, logger)

	err := svc.UpdateCategoryPattern(context.Background(), user.ID(), 999999, "netflix")
	if err == nil {
		t.Fatal("Expected error when category does not exist, got nil")
	}
}

func TestServiceGetUncategorized_GroupsByDescription(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "coffee shop", "USD", -500, time.Now(), domain.ChargeType, nil),
		domain.NewExpense(0, "Test Source", "coffee shop", "USD", -600, time.Now(), domain.ChargeType, nil),
		domain.NewExpense(0, "Test Source", "hardware store", "USD", -1500, time.Now(), domain.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)

	grouped, keys, totalExpenses, totalAmount, err := svc.GetUncategorized(context.Background(), user.ID(), "")
	if err != nil {
		t.Fatalf("GetUncategorized returned error: %v", err)
	}

	if totalExpenses != 3 {
		t.Fatalf("Expected 3 total expenses, got %d", totalExpenses)
	}

	if totalAmount != -2600 {
		t.Fatalf("Expected total amount -2600, got %d", totalAmount)
	}

	if len(keys) != 2 {
		t.Fatalf("Expected 2 grouping keys, got %d (%v)", len(keys), keys)
	}

	coffeeInfo, ok := grouped["coffee shop"]
	if !ok {
		t.Fatal("Expected 'coffee shop' group to exist")
	}

	if coffeeInfo.Count != 2 {
		t.Fatalf("Expected coffee shop count to be 2, got %d", coffeeInfo.Count)
	}

	if coffeeInfo.Total != -1100 {
		t.Fatalf("Expected coffee shop total to be -1100, got %d", coffeeInfo.Total)
	}

	if coffeeInfo.Slug != "coffee-shop" {
		t.Fatalf("Expected slug 'coffee-shop', got %s", coffeeInfo.Slug)
	}

	// keys should be sorted by count descending
	if keys[0] != "coffee shop" {
		t.Fatalf("Expected 'coffee shop' to be the first key (highest count), got %s", keys[0])
	}
}

func TestServiceTest_ReturnsMatchesWithoutWriting(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "cinema", "USD", -123456, time.Now(), domain.ChargeType, nil),
		domain.NewExpense(0, "Test Source", "grocery", "USD", -2000, time.Now(), domain.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)

	matched, err := svc.Test(context.Background(), user.ID(), "cinema|movie", domain.ChargeCategory)
	if err != nil {
		t.Fatalf("Test returned error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("Expected 1 matched expense, got %d", len(matched))
	}

	if matched[0].Description() != "cinema" {
		t.Fatalf("Expected matched expense to be 'cinema', got %s", matched[0].Description())
	}

	// Verify no writes happened - expenses should remain uncategorized
	allExpenses, err := s.GetExpenses(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	for _, ex := range allExpenses {
		if ex.CategoryID() != nil {
			t.Fatalf("Expected expense %s to remain uncategorized, got category %d", ex.Description(), *ex.CategoryID())
		}
	}

	// Also verify no category was created
	categories, err := s.GetCategories(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get categories: %v", err)
	}

	for _, c := range categories {
		if c.Name() != domain.ExcludeCategory {
			t.Fatalf("Expected no categories to be created, found %s", c.Name())
		}
	}
}

func TestServiceCreate_IncomeCategoryOnlyCategorizesIncome(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	ctx := context.Background()

	_, err := s.InsertExpenses(ctx, user.ID(), []domain.Expense{
		domain.NewExpense(0, "bank", "acme payroll", "EUR", 300000, time.Now(), domain.IncomeType, nil),
		domain.NewExpense(0, "bank", "acme store", "EUR", -2000, time.Now(), domain.ChargeType, nil),
	})
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)

	categoryID, matched, err := svc.Create(ctx, user.ID(), domain.CategoryFormData{
		Name:          "Salary",
		Pattern:       "acme",
		MonthlyBudget: 50000,
		Type:          domain.IncomeCategory,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if len(matched) != 1 || matched[0].Description() != "acme payroll" {
		t.Fatalf("Expected only the income expense to be categorized, got %d", len(matched))
	}

	created, err := s.GetCategory(ctx, user.ID(), categoryID)
	if err != nil {
		t.Fatalf("Failed to get category: %v", err)
	}
	if created.Type() != domain.IncomeCategory {
		t.Errorf("Type = %v, want income", created.Type())
	}
	if created.MonthlyBudget() != 0 {
		t.Errorf("MonthlyBudget = %d, want 0 for income category", created.MonthlyBudget())
	}

	// The test preview must agree with Create.
	preview, err := svc.Test(ctx, user.ID(), "acme", domain.ChargeCategory)
	if err != nil {
		t.Fatalf("Test returned error: %v", err)
	}
	if len(preview) != 1 || preview[0].Description() != "acme store" {
		t.Fatalf("Expected preview to only include the charge, got %d", len(preview))
	}
}

func TestServiceUpdate_TypeChange(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	ctx := context.Background()

	usedID, err := s.CreateCategory(ctx, user.ID(), "Food", "food", 10000, domain.ChargeCategory)
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}
	emptyID, err := s.CreateCategory(ctx, user.ID(), "Salary", "payroll", 10000, domain.ChargeCategory)
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	_, err = s.InsertExpenses(ctx, user.ID(), []domain.Expense{
		domain.NewExpense(0, "bank", "food shop", "EUR", -1000, time.Now(), domain.ChargeType, &usedID),
		domain.NewExpense(0, "bank", "payroll", "EUR", 300000, time.Now(), domain.IncomeType, nil),
	})
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)

	// A category holding charges cannot become an income category.
	if _, _, _, err = svc.Update(ctx, user.ID(), usedID, "", "", "", "income"); err == nil {
		t.Fatal("Expected error changing type of a category holding charges")
	}

	// An empty one can; its budget is dropped and income gets swept in.
	updated, changed, matchingChanged, err := svc.Update(ctx, user.ID(), emptyID, "", "", "", "income")
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if !changed || !matchingChanged {
		t.Fatalf("Expected changed and matchingChanged, got %v %v", changed, matchingChanged)
	}
	if updated.Type() != domain.IncomeCategory || updated.MonthlyBudget() != 0 {
		t.Fatalf("Expected income category without budget, got %v %d", updated.Type(), updated.MonthlyBudget())
	}

	byCategory, err := s.GetExpensesByCategory(ctx, user.ID(), emptyID)
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}
	if len(byCategory) != 1 || byCategory[0].Description() != "payroll" {
		t.Fatalf("Expected payroll to be recategorized into Salary, got %d expenses", len(byCategory))
	}
}

func TestServiceUpdate_ExcludeCategoryKeepsAnyType(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	ctx := context.Background()

	exclude, err := s.GetExcludeCategory(ctx, user.ID())
	if err != nil {
		t.Fatalf("Failed to get exclude category: %v", err)
	}

	svc := New(s, logger)

	updated, _, _, err := svc.Update(ctx, user.ID(), exclude.ID(), "", "", "", "charge")
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Type() != domain.AnyCategory {
		t.Fatalf("Type = %v, want any", updated.Type())
	}
}

func TestServiceUpdateCategoryPattern_SkipsOtherType(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	ctx := context.Background()

	categoryID, err := s.CreateCategory(ctx, user.ID(), "Shopping", "shop", 0, domain.ChargeCategory)
	if err != nil {
		t.Fatalf("Failed to create Category: %v", err)
	}

	_, err = s.InsertExpenses(ctx, user.ID(), []domain.Expense{
		domain.NewExpense(0, "bank", "amazon", "EUR", -1000, time.Now(), domain.ChargeType, nil),
		domain.NewExpense(0, "bank", "amazon", "EUR", 1000, time.Now(), domain.IncomeType, nil),
	})
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)
	if err = svc.UpdateCategoryPattern(ctx, user.ID(), categoryID, "amazon"); err != nil {
		t.Fatalf("UpdateCategoryPattern returned error: %v", err)
	}

	byCategory, err := s.GetExpensesByCategory(ctx, user.ID(), categoryID)
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}
	if len(byCategory) != 1 || byCategory[0].Type() != domain.ChargeType {
		t.Fatalf("Expected only the charge in Shopping, got %d expenses", len(byCategory))
	}
}
