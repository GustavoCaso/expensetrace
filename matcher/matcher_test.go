package matcher

import (
	"testing"

	"github.com/GustavoCaso/expensetrace/domain"
)

func TestNew(t *testing.T) {
	categories := []domain.Category{
		domain.NewCategory(1, "Food", "restaurant|food|grocery", 0, domain.ChargeCategory),
		domain.NewCategory(2, "Transport", "uber|taxi|transit", 0, domain.ChargeCategory),
	}

	matcher := New(categories)

	if len(matcher.matchers) != 2 {
		t.Errorf("Expected 2 matchers, got %d", len(matcher.matchers))
	}

	if len(matcher.categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(matcher.categories))
	}
}

func TestMatch(t *testing.T) {
	foodCategoryID := int64(1)
	transportCategoryID := int64(1)
	entertaimentCategoryID := int64(1)

	categories := []domain.Category{
		domain.NewCategory(foodCategoryID, "Food", "restaurant|food|grocery", 0, domain.ChargeCategory),
		domain.NewCategory(transportCategoryID, "Transport", "uber|taxi|transit", 0, domain.ChargeCategory),
		domain.NewCategory(entertaimentCategoryID, "Entertainment", "netflix|spotify|movie", 0, domain.ChargeCategory),
	}

	matcher := New(categories)

	tests := []struct {
		name     string
		input    string
		wantID   *int64
		wantName string
	}{
		{
			name:     "should match food category",
			input:    "restaurant bill",
			wantID:   &foodCategoryID,
			wantName: "Food",
		},
		{
			name:     "should match transport category",
			input:    "uber ride",
			wantID:   &transportCategoryID,
			wantName: "Transport",
		},
		{
			name:     "should match entertainment category",
			input:    "netflix subscription",
			wantID:   &entertaimentCategoryID,
			wantName: "Entertainment",
		},
		{
			name:     "should not match any category",
			input:    "random text",
			wantID:   nil,
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotName := matcher.Match(tt.input, domain.ChargeType)

			if gotID == nil {
				if gotID != tt.wantID {
					t.Errorf("Match() ID = %v, want %v", gotID, tt.wantID)
				}
			} else {
				if *gotID != *tt.wantID {
					t.Errorf("Match() ID = %v, want %v", gotID, tt.wantID)
				}
			}

			if gotName != tt.wantName {
				t.Errorf("Match() Name = %v, want %v", gotName, tt.wantName)
			}
		})
	}
}

func TestCategories(t *testing.T) {
	categories := []domain.Category{
		domain.NewCategory(1, "Food", "restaurant|food|grocery", 0, domain.ChargeCategory),
		domain.NewCategory(2, "Transport", "uber|taxi|transit", 0, domain.ChargeCategory),
	}

	matcher := New(categories)
	got := matcher.Categories()

	if len(got) != len(categories) {
		t.Errorf("Categories() returned %d categories, want %d", len(got), len(categories))
	}

	for i, cat := range got {
		if cat.ID() != categories[i].ID() {
			t.Errorf("Categories()[%d].ID = %v, want %v", i, cat.ID(), categories[i].ID())
		}
		if cat.Name() != categories[i].Name() {
			t.Errorf("Categories()[%d].Name = %v, want %v", i, cat.Name(), categories[i].Name())
		}
		if cat.Pattern() != categories[i].Pattern() {
			t.Errorf("Categories()[%d].Pattern = %v, want %v", i, cat.Pattern(), categories[i].Pattern())
		}
	}
}

func TestMatchRespectsCategoryType(t *testing.T) {
	refundsID := int64(1)
	shoppingID := int64(2)
	excludeID := int64(3)

	m := New([]domain.Category{
		domain.NewCategory(refundsID, "Refunds", "amazon", 0, domain.IncomeCategory),
		domain.NewCategory(shoppingID, "Shopping", "amazon", 0, domain.ChargeCategory),
		domain.NewCategory(excludeID, domain.ExcludeCategory, "transfer", 0, domain.AnyCategory),
	})

	tests := []struct {
		name        string
		input       string
		expenseType domain.ExpenseType
		wantID      *int64
	}{
		{"charge skips income category", "amazon order", domain.ChargeType, &shoppingID},
		{"income skips charge category", "amazon refund", domain.IncomeType, &refundsID},
		{"any category matches charge", "transfer out", domain.ChargeType, &excludeID},
		{"any category matches income", "transfer in", domain.IncomeType, &excludeID},
		{"no match", "coffee", domain.IncomeType, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, _ := m.Match(tt.input, tt.expenseType)
			switch {
			case tt.wantID == nil && gotID != nil:
				t.Errorf("Match() = %d, want nil", *gotID)
			case tt.wantID != nil && (gotID == nil || *gotID != *tt.wantID):
				t.Errorf("Match() = %v, want %d", gotID, *tt.wantID)
			}
		})
	}
}
