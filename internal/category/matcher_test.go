package category

import (
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/db"
)

func TestNewMatcher(t *testing.T) {
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
	}

	matcher := NewMatcher(categories)

	if len(matcher.matchers) != 2 {
		t.Errorf("Expected 2 matchers, got %d", len(matcher.matchers))
	}

	if len(matcher.categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(matcher.categories))
	}
}

func TestMatch(t *testing.T) {
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
		{ID: 3, Name: "Entertainment", Pattern: "netflix|spotify|movie"},
	}

	matcher := NewMatcher(categories)

	tests := []struct {
		name     string
		input    string
		wantID   int
		wantName string
	}{
		{
			name:     "should match food category",
			input:    "restaurant bill",
			wantID:   1,
			wantName: "Food",
		},
		{
			name:     "should match transport category",
			input:    "uber ride",
			wantID:   2,
			wantName: "Transport",
		},
		{
			name:     "should match entertainment category",
			input:    "netflix subscription",
			wantID:   3,
			wantName: "Entertainment",
		},
		{
			name:     "should not match any category",
			input:    "random text",
			wantID:   0,
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotName := matcher.Match(tt.input)
			if gotID != tt.wantID {
				t.Errorf("Match() ID = %v, want %v", gotID, tt.wantID)
			}
			if gotName != tt.wantName {
				t.Errorf("Match() Name = %v, want %v", gotName, tt.wantName)
			}
		})
	}
}

func TestCategories(t *testing.T) {
	categories := []db.Category{
		{ID: 1, Name: "Food", Pattern: "restaurant|food|grocery"},
		{ID: 2, Name: "Transport", Pattern: "uber|taxi|transit"},
	}

	matcher := NewMatcher(categories)
	got := matcher.Categories()

	if len(got) != len(categories) {
		t.Errorf("Categories() returned %d categories, want %d", len(got), len(categories))
	}

	for i, cat := range got {
		if cat.ID != categories[i].ID {
			t.Errorf("Categories()[%d].ID = %v, want %v", i, cat.ID, categories[i].ID)
		}
		if cat.Name != categories[i].Name {
			t.Errorf("Categories()[%d].Name = %v, want %v", i, cat.Name, categories[i].Name)
		}
		if cat.Pattern != categories[i].Pattern {
			t.Errorf("Categories()[%d].Pattern = %v, want %v", i, cat.Pattern, categories[i].Pattern)
		}
	}
}
