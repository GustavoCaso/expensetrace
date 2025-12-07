package matcher

import (
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func TestNew(t *testing.T) {
	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery", 0),
		storage.NewCategory(2, "Transport", "uber|taxi|transit", 0),
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

	categories := []storage.Category{
		storage.NewCategory(foodCategoryID, "Food", "restaurant|food|grocery", 0),
		storage.NewCategory(transportCategoryID, "Transport", "uber|taxi|transit", 0),
		storage.NewCategory(entertaimentCategoryID, "Entertainment", "netflix|spotify|movie", 0),
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
			gotID, gotName := matcher.Match(tt.input)

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
	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery", 0),
		storage.NewCategory(2, "Transport", "uber|taxi|transit", 0),
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
