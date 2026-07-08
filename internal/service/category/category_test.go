package category

import (
	"context"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/domain"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

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

	categoryID, err := s.CreateCategory(context.Background(), user.ID(), "Entertainment", "restaurant|bars|cinema", 0)
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

	categoryID, err := s.CreateCategory(context.Background(), user.ID(), "Entertainment", "restaurant|bars|cinema", 0)
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

	matched, err := svc.Test(context.Background(), user.ID(), "cinema|movie")
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
