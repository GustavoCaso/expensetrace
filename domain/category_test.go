package domain

import "testing"

func TestCategoryTypeAccepts(t *testing.T) {
	tests := []struct {
		categoryType CategoryType
		expenseType  ExpenseType
		want         bool
	}{
		{ChargeCategory, ChargeType, true},
		{ChargeCategory, IncomeType, false},
		{IncomeCategory, ChargeType, false},
		{IncomeCategory, IncomeType, true},
		{AnyCategory, ChargeType, true},
		{AnyCategory, IncomeType, true},
	}

	for _, tt := range tests {
		if got := tt.categoryType.Accepts(tt.expenseType); got != tt.want {
			t.Errorf("%s.Accepts(%d) = %v, want %v", tt.categoryType, tt.expenseType, got, tt.want)
		}
	}
}

func TestParseCategoryType(t *testing.T) {
	tests := []struct {
		in      string
		want    CategoryType
		wantErr bool
	}{
		{"", ChargeCategory, false},
		{"charge", ChargeCategory, false},
		{"income", IncomeCategory, false},
		{"any", ChargeCategory, true}, // not user-selectable
		{"0", ChargeCategory, true},
	}

	for _, tt := range tests {
		got, err := ParseCategoryType(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseCategoryType(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
		}
		if err == nil && got != tt.want {
			t.Errorf("ParseCategoryType(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
