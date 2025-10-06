package importutil

import (
	"strings"
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func TestFieldMappingValidate(t *testing.T) {
	tests := []struct {
		name        string
		mapping     *FieldMapping
		headerCount int
		wantErr     bool
		errContains string
	}{
		{
			name: "valid mapping",
			mapping: &FieldMapping{
				Source:            "Test Bank",
				DateColumn:        0,
				DescriptionColumn: 1,
				AmountColumn:      2,
				CurrencyColumn:    3,
			},
			headerCount: 4,
			wantErr:     false,
		},
		{
			name: "missing source",
			mapping: &FieldMapping{
				Source:            "",
				DateColumn:        0,
				DescriptionColumn: 1,
				AmountColumn:      2,
				CurrencyColumn:    3,
			},
			headerCount: 4,
			wantErr:     true,
			errContains: "source",
		},
		{
			name: "negative column index",
			mapping: &FieldMapping{
				Source:            "Test Bank",
				DateColumn:        -1,
				DescriptionColumn: 1,
				AmountColumn:      2,
				CurrencyColumn:    3,
			},
			headerCount: 4,
			wantErr:     true,
			errContains: "date column",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mapping.Validate(tt.headerCount)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.errContains)
			}
		})
	}
}

func TestApplyMapping(t *testing.T) {
	csvData := `date,description,amount,currency
01/01/2024,restaurant bill,-50.00,USD
02/01/2024,uber ride,-25.00,USD
03/01/2024,salary,2500.00,USD`

	reader := strings.NewReader(csvData)
	parsed, err := ParseFile("test.csv", reader)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	mapping := &FieldMapping{
		Source:            "Test Bank",
		DateColumn:        0,
		DescriptionColumn: 1,
		AmountColumn:      2,
		CurrencyColumn:    3,
	}

	catFoodID := int64(1)
	catTransportID := int64(2)
	categories := []storage.Category{
		storage.NewCategory(catFoodID, "Food", "restaurant|food|grocery"),
		storage.NewCategory(catTransportID, "Transport", "uber|taxi|transit"),
	}
	categoryMatcher := matcher.New(categories)

	result, err := ApplyMapping(parsed, mapping, categoryMatcher)
	if err != nil {
		t.Fatalf("ApplyMapping failed: %v", err)
	}

	if len(result.Expenses) != 3 {
		t.Fatalf("Expected 3 expenses, got %d", len(result.Expenses))
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}

	exp1 := result.Expenses[0]
	if exp1.Source() != "Test Bank" {
		t.Errorf("Expense[0].Source = %q, want 'Test Bank'", exp1.Source())
	}
	if exp1.Description() != "restaurant bill" {
		t.Errorf("Expense[0].Description = %q, want 'restaurant bill'", exp1.Description())
	}
	if exp1.Amount() != -5000 {
		t.Errorf("Expense[0].Amount = %d, want -5000", exp1.Amount())
	}
	if exp1.Type() != storage.ChargeType {
		t.Errorf("Expense[0].Type = %v, want ChargeType", exp1.Type())
	}
	if *exp1.CategoryID() != catFoodID {
		t.Errorf("Expense[0].CategoryID = %d, want %d", *exp1.CategoryID(), catFoodID)
	}

	exp2 := result.Expenses[1]
	if *exp2.CategoryID() != catTransportID {
		t.Errorf("Expense[1].Category = %d, want %d", *exp2.CategoryID(), catTransportID)
	}

	exp3 := result.Expenses[2]
	if exp3.Amount() != 250000 {
		t.Errorf("Expense[2].Amount = %d, want 250000", exp3.Amount())
	}
	if exp3.Type() != storage.IncomeType {
		t.Errorf("Expense[2].Type = %v, want IncomeType", exp3.Type())
	}
	if exp3.CategoryID() != nil {
		t.Errorf("Expense[2].Category = <nil>, got: %d", *exp3.CategoryID())
	}
}

func TestApplyMappingWithErrors(t *testing.T) {
	csvData := `date,description,amount,currency
invalid-date,Coffee,-5.00,USD
02/01/2024,Lunch,-12.00,USD
03/01/2024,Dinner,invalid-amount,USD`

	reader := strings.NewReader(csvData)
	parsed, err := ParseFile("test.csv", reader)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	mapping := &FieldMapping{
		Source:            "Test Bank",
		DateColumn:        0,
		DescriptionColumn: 1,
		AmountColumn:      2,
		CurrencyColumn:    3,
	}

	categories := []storage.Category{}
	categoryMatcher := matcher.New(categories)

	result, err := ApplyMapping(parsed, mapping, categoryMatcher)
	if err != nil {
		t.Fatalf("ApplyMapping failed: %v", err)
	}

	if len(result.Expenses) != 1 {
		t.Errorf("Expected 1 successful expense, got %d", len(result.Expenses))
	}
	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(result.Errors))
	}

	if result.Expenses[0].Source() != "Test Bank" {
		t.Errorf("Successful expense source = %q, want 'Test Bank'", result.Expenses[0].Source())
	}

	if result.Errors[0].RowIndex != 0 {
		t.Errorf("Error[0].RowIndex = %d, want 0", result.Errors[0].RowIndex)
	}
	if result.Errors[1].RowIndex != 2 {
		t.Errorf("Error[1].RowIndex = %d, want 2", result.Errors[1].RowIndex)
	}
}

func TestParseDateWithDifferentFormats(t *testing.T) {
	tests := []struct {
		dateStr string
		wantErr bool
	}{
		{"01/01/2024", false},
		{"01/01/2024", false},
		{"2024-01-01", false},
		{"2024-01-01T10:30:00Z", false},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.dateStr, func(t *testing.T) {
			_, err := parseDate(tt.dateStr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseDate(%q) expected error got nil", tt.dateStr)
					return
				}
				return
			}

			if err != nil {
				t.Errorf("parseDate(%q) expected no error got: %v", tt.dateStr, err)
			}
		})
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		amountStr string
		want      int64
		wantErr   bool
	}{
		{"-50.00", -5000, false},
		{"50.00", 5000, false},
		{"-5000", -5000, false},
		{"2500.50", 250050, false},
		{"-12.5", -125, false},
		{"100", 100, false},
		{"-1,234.56", -123456, false}, // With comma separator
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.amountStr, func(t *testing.T) {
			got, err := parseAmount(tt.amountStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAmount(%q) error = %v, wantErr %v", tt.amountStr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseAmount(%q) = %d, want %d", tt.amountStr, got, tt.want)
			}
		})
	}
}

func TestApplyMappingInvalidMapping(t *testing.T) {
	csvData := `date,description,amount,currency
01/01/2024,Coffee,-5.00,USD`

	reader := strings.NewReader(csvData)
	parsed, err := ParseFile("test.csv", reader)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Invalid mapping with column index out of range
	mapping := &FieldMapping{
		Source:            "Test Bank",
		DateColumn:        10,
		DescriptionColumn: 1,
		AmountColumn:      2,
		CurrencyColumn:    3,
	}

	categories := []storage.Category{}
	categoryMatcher := matcher.New(categories)

	_, err = ApplyMapping(parsed, mapping, categoryMatcher)
	if err == nil {
		t.Fatal("Expected error for invalid mapping")
	}
	if !strings.Contains(err.Error(), "invalid mapping") {
		t.Errorf("Expected 'invalid mapping' error, got: %v", err)
	}
}
