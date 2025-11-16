package importutil

import (
	"context"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestImportCSV(t *testing.T) {
	tests := []struct {
		name                 string
		filename             string
		csvData              string
		expectedSource       string
		expectedCurrency     string
		expectedAmounts      []int64
		expectedDescriptions []string
	}{
		{
			name:     "Evo CSV format",
			filename: "evo_test.csv",
			csvData: `Fecha de la operación,Fecha Valor,Concepto,Importe,Divisa,Tipo de movimiento,Saldo disponible
01/01/2024,,Restaurant bill,-1234.56,USD,,5000.00
02/01/2024,,Uber ride,-5000.00,USD,,0.00
03/01/2024,,Salary,500000.00,USD,,500000.00`,
			expectedSource:       "Evo",
			expectedCurrency:     "USD",
			expectedAmounts:      []int64{-123456, -500000, 50000000},
			expectedDescriptions: []string{"restaurant bill", "uber ride", "salary"},
		},
		{
			name:     "Bankinter CSV format",
			filename: "bankinter_test.csv",
			csvData: `Fecha Contable,Fecha Valor,Descripcion,Importe,Saldo,Columna6,Columna7
01/01/2024,01/01/2024,Restaurant bill,"-1,234.56",5000.00,,
02/01/2024,02/01/2024,Uber ride,"-5,000.00",0.00,,
03/01/2024,03/01/2024,Salary,"500,000.00",500000.00,,`,
			expectedSource:       "Bankinter",
			expectedCurrency:     "EUR",
			expectedAmounts:      []int64{-123456, -500000, 50000000},
			expectedDescriptions: []string{"restaurant bill", "uber ride", "salary"},
		},
		{
			name:     "Revolut CSV format",
			filename: "revolut_test.csv",
			csvData: `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CHARGE,Current,2024-01-01 10:00:00,2024-01-01 10:01:00,Restaurant bill,1234.56,0.00,USD,COMPLETED,5000.00
CHARGE,Current,2024-01-02 11:00:00,2024-01-02 11:01:00,Uber ride,5000.00,0.00,USD,COMPLETED,0.00
INCOME,Current,2024-01-03 12:00:00,2024-01-03 12:01:00,Salary,500000.00,0.00,USD,COMPLETED,500000.00`,
			expectedSource:       "Revolut",
			expectedCurrency:     "USD",
			expectedAmounts:      []int64{-123456, -500000, 50000000},
			expectedDescriptions: []string{"restaurant bill", "uber ride", "salary"},
		},
		{
			name:     "Revolut CSV with fees",
			filename: "revolut_fees.csv",
			csvData: `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CHARGE,Current,2024-01-01 10:00:00,2024-01-01 10:01:00,Restaurant bill,1234.56,50.00,USD,COMPLETED,5000.00
CHARGE,Current,2024-01-02 11:00:00,2024-01-02 11:01:00,Uber ride,5000.00,100.50,USD,COMPLETED,0.00
INCOME,Current,2024-01-03 12:00:00,2024-01-03 12:01:00,Salary,500000.00,0.00,USD,COMPLETED,500000.00`,
			expectedSource:       "Revolut",
			expectedCurrency:     "USD",
			expectedAmounts:      []int64{-118456, -489950, 50000000},
			expectedDescriptions: []string{"restaurant bill", "uber ride", "salary"},
		},
		{
			name:     "Evo card payment string removal",
			filename: "evo_special.csv",
			csvData: `Fecha de la operación,Fecha Valor,Concepto,Importe,Divisa,Tipo de movimiento,Saldo disponible
01/01/2024,,Pago en el dia TJ-Amazon Purchase,-50.00,EUR,,5000.00`,
			expectedSource:       "Evo",
			expectedCurrency:     "EUR",
			expectedAmounts:      []int64{-5000},
			expectedDescriptions: []string{"amazon purchase"},
		},
		{
			name:     "Revolut single transaction with fee",
			filename: "revolut_single_fee.csv",
			csvData: `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CHARGE,Current,2024-01-01 10:00:00,2024-01-01 10:01:00,ATM Withdrawal,100.00,2.50,USD,COMPLETED,5000.00`,
			expectedSource:   "Revolut",
			expectedCurrency: "USD",
			expectedAmounts:  []int64{-9750},
			expectedDescriptions: []string{
				"atm withdrawal",
			},
		},
		{
			name:     "Bankinter large amount with commas",
			filename: "bankinter_large.csv",
			csvData: `Fecha Contable,Fecha Valor,Descripcion,Importe,Saldo,Columna6,Columna7
01/01/2024,01/01/2024,Large Purchase,"-12,345.67",5000.00,,`,
			expectedSource:       "Bankinter",
			expectedCurrency:     "EUR",
			expectedAmounts:      []int64{-1234567},
			expectedDescriptions: []string{"large purchase"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.TestLogger(t)
			s, user := testutil.SetupTestStorage(t, logger)

			// Create test categories
			categories := []storage.Category{
				storage.NewCategory(1, "Food", "restaurant|food|grocery"),
				storage.NewCategory(2, "Transport", "uber|taxi|transit"),
				storage.NewCategory(3, "Shopping", "amazon|purchase"),
				storage.NewCategory(4, "Cash", "atm|withdrawal"),
			}

			for _, c := range categories {
				_, err := s.CreateCategory(context.Background(), user.ID(), c.Name(), c.Pattern())
				if err != nil {
					t.Fatalf("Failed to create category: %v", err)
				}
			}
			matcher := matcher.New(categories)

			reader := strings.NewReader(tt.csvData)
			info := ImportCSV(context.Background(), user.ID(), tt.filename, reader, s, matcher)
			if info.Error != nil {
				t.Errorf("Import failed with error: %v", info.Error)
			}

			// Verify imported expenses
			expenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
			if err != nil {
				t.Fatalf("Failed to get expenses: %v", err)
			}

			expectedCount := len(tt.expectedAmounts)
			if len(expenses) != expectedCount {
				t.Fatalf("Expected %d expenses, got %d", expectedCount, len(expenses))
			}

			// Verify all expenses amounts, source, and currency
			for i, expectedAmount := range tt.expectedAmounts {
				if expenses[i].Source() != tt.expectedSource {
					t.Errorf("Expense[%d].Source = %v, want %v", i, expenses[i].Source(), tt.expectedSource)
				}
				if expenses[i].Amount() != expectedAmount {
					t.Errorf("Expense[%d].Amount = %v, want %v", i, expenses[i].Amount(), expectedAmount)
				}
				if expenses[i].Currency() != tt.expectedCurrency {
					t.Errorf("Expense[%d].Currency = %v, want %v", i, expenses[i].Currency(), tt.expectedCurrency)
				}

				if expenses[i].Description() != tt.expectedDescriptions[i] {
					t.Errorf(
						"Expense[%d].Description = %v, want %v",
						i,
						expenses[i].Description(),
						tt.expectedDescriptions[i],
					)
				}

				// Verify expense type based on amount
				expectedType := storage.ChargeType
				if expectedAmount >= 0 {
					expectedType = storage.IncomeType
				}
				if expenses[i].Type() != expectedType {
					t.Errorf("Expense[%d].Type = %v, want %v", i, expenses[i].Type(), expectedType)
				}
			}
		})
	}
}

func TestImportJSON(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	// Create test categories
	categories := []storage.Category{
		storage.NewCategory(1, "Food", "restaurant|food|grocery"),
		storage.NewCategory(2, "Transport", "uber|taxi|transit"),
	}

	for _, c := range categories {
		_, err := s.CreateCategory(context.Background(), user.ID(), c.Name(), c.Pattern())
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}
	}

	matcher := matcher.New(categories)

	// Test JSON data
	jsonData := `[
		{
			"source": "Test Source",
			"date": "2024-01-01T00:00:00Z",
			"description": "Restaurant bill",
			"amount": -123456,
			"currency": "USD"
		},
		{
			"source": "Test Source",
			"date": "2024-01-02T00:00:00Z",
			"description": "Uber ride",
			"amount": -500000,
			"currency": "USD"
		},
		{
			"source": "Test Source",
			"date": "2024-01-03T00:00:00Z",
			"description": "Salary",
			"amount": 50000000,
			"currency": "USD"
		}
	]`

	reader := strings.NewReader(jsonData)
	valid, jsonExpenses := SupportedJSONSchema(reader)
	if !valid {
		t.Fatal("JSON expenses are invalid")
	}

	info := ImportJSON(context.Background(), user.ID(), jsonExpenses, s, matcher)

	if info.Error != nil {
		t.Errorf("Import failed with error: %v", info.Error)
	}

	// Verify imported expenses
	expenses, err := s.GetAllExpenseTypes(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	if len(expenses) != 3 {
		t.Fatalf("Expected 3 expenses, got %d", len(expenses))
	}

	// Verify first expense
	if expenses[0].Source() != "Test Source" {
		t.Errorf("Expense[0].Source = %v, want Test Source", expenses[0].Source())
	}
	if expenses[0].Description() != "restaurant bill" {
		t.Errorf("Expense[0].Description = %v, want restaurant bill", expenses[0].Description())
	}
	if expenses[0].Amount() != -123456 {
		t.Errorf("Expense[0].Amount = %v, want -123456", expenses[0].Amount())
	}
	if expenses[0].Type() != storage.ChargeType {
		t.Errorf("Expense[0].Type = %v, want ChargeType", expenses[0].Type())
	}
	if expenses[0].Currency() != "USD" {
		t.Errorf("Expense[0].Currency = %v, want USD", expenses[0].Currency())
	}
	if expenses[0].CategoryID() == nil || *expenses[0].CategoryID() != 1 {
		t.Errorf("Expense[0].CategoryID = %v, want 1", expenses[0].CategoryID())
	}

	// Verify second expense
	if expenses[1].Description() != "uber ride" {
		t.Errorf("Expense[1].Description = %v, want uber ride", expenses[1].Description())
	}
	if expenses[1].Amount() != -500000 {
		t.Errorf("Expense[1].Amount = %v, want -500000", expenses[1].Amount())
	}
	if expenses[1].CategoryID() == nil || *expenses[1].CategoryID() != 2 {
		t.Errorf("Expense[1].CategoryID = %v, want 2", expenses[1].CategoryID())
	}

	// Verify third expense
	if expenses[2].Amount() != 50000000 {
		t.Errorf("Expense[2].Amount = %v, want 50000000", expenses[2].Amount())
	}
	if expenses[2].Type() != storage.IncomeType {
		t.Errorf("Expense[2].Type = %v, want IncomeType", expenses[2].Type())
	}
}

func TestImportInvalidCSV(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		csvData  string
	}{
		{
			name:     "Evo CSV with invalid date",
			filename: "evo_test.csv",
			csvData: `Fecha de la operación,Fecha Valor,Concepto,Importe,Divisa,Tipo de movimiento,Saldo disponible
invalid-date,,Restaurant bill,-1234.56,USD,,5000.00`,
		},
		{
			name:     "Bankinter CSV with invalid date",
			filename: "bankinter_test.csv",
			csvData: `Fecha Contable,Fecha Valor,Descripcion,Importe,Saldo,Columna6,Columna7
invalid-date,01/01/2024,Restaurant bill,"-1,234.56",5000.00,,`,
		},
		{
			name:     "Revolut CSV with invalid date",
			filename: "revolut_test.csv",
			csvData: `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CHARGE,Current,invalid-date,2024-01-01 10:01:00,Restaurant bill,1234.56,0.00,USD,COMPLETED,5000.00`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutil.TestLogger(t)
			s, user := testutil.SetupTestStorage(t, logger)

			// Create test categories
			categories := []storage.Category{
				storage.NewCategory(1, "Food", "restaurant|food|grocery"),
				storage.NewCategory(2, "Transport", "uber|taxi|transit"),
			}
			matcher := matcher.New(categories)

			reader := strings.NewReader(tt.csvData)
			info := ImportCSV(context.Background(), user.ID(), tt.filename, reader, s, matcher)
			if info.Error == nil {
				t.Errorf("Expected error")
			}
			if !strings.Contains(info.Error.Error(), "parsing time") {
				t.Errorf("Expected parsing error, got: %v", info.Error)
			}
		})
	}
}

func TestImportInvalidJSON(t *testing.T) {
	// Test with invalid JSON data
	jsonData := `[
		{
			"source": "Test Source",
			"date": "invalid-date",
			"description": "Restaurant bill",
			"amount": -123456,
			"currency": "USD"
		}
	]`

	reader := strings.NewReader(jsonData)

	valid, _ := SupportedJSONSchema(reader)
	if valid {
		t.Fatal("expected invalid JSON")
	}
}
