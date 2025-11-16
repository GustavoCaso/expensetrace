package export

import (
	"bytes"
	"context"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestCSV(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	// Create test categories
	ctx := context.Background()
	categoryID1, err := s.CreateCategory(ctx, user.ID(), "Food", "restaurant|food")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	categoryID2, err := s.CreateCategory(ctx, user.ID(), "Transport", "uber|taxi")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	// Create test expenses
	testDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	expenses := []storage.Expense{
		storage.NewExpense(
			0,
			"TestSource",
			"restaurant bill",
			"USD",
			-5000,
			testDate,
			storage.ChargeType,
			&categoryID1,
		),
		storage.NewExpense(0, "TestSource", "uber ride", "EUR", -3000, testDate, storage.ChargeType, &categoryID2),
		storage.NewExpense(0, "TestSource", "salary", "USD", 500000, testDate, storage.IncomeType, nil),
	}

	_, err = s.InsertExpenses(ctx, user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert expenses: %v", err)
	}

	// Get all expenses for export
	allExpenses, err := s.GetAllExpenseTypes(ctx, user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	// Export to CSV
	var buf bytes.Buffer
	err = CSV(ctx, user.ID(), &buf, allExpenses, s)
	if err != nil {
		t.Fatalf("CSV failed: %v", err)
	}

	// Parse the CSV output
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Verify header
	if len(records) < 1 {
		t.Fatal("Expected at least header row in CSV")
	}

	expectedHeader := []string{"ID", "Source", "Date", "Description", "Amount", "Type", "Currency", "Category"}
	header := records[0]
	if len(header) != len(expectedHeader) {
		t.Fatalf("Expected %d columns in header, got %d", len(expectedHeader), len(header))
	}

	for i, col := range expectedHeader {
		if header[i] != col {
			t.Errorf("Header[%d] = %v, want %v", i, header[i], col)
		}
	}

	// Verify we have 4 rows (1 header + 3 expenses)
	if len(records) != 4 {
		t.Fatalf("Expected 4 rows (header + 3 expenses), got %d", len(records))
	}

	// Verify first expense (restaurant bill)
	row1 := records[1]
	if row1[1] != "TestSource" {
		t.Errorf("Row1[Source] = %v, want TestSource", row1[1])
	}
	if row1[2] != "2024-01-15" {
		t.Errorf("Row1[Date] = %v, want 2024-01-15", row1[2])
	}
	if row1[3] != "restaurant bill" {
		t.Errorf("Row1[Description] = %v, want restaurant bill", row1[3])
	}
	if row1[4] != "-50.00" {
		t.Errorf("Row1[Amount] = %v, want -50.00", row1[4])
	}
	if row1[5] != "charge" {
		t.Errorf("Row1[Type] = %v, want charge", row1[5])
	}
	if row1[6] != "USD" {
		t.Errorf("Row1[Currency] = %v, want USD", row1[6])
	}
	if row1[7] != "Food" {
		t.Errorf("Row1[Category] = %v, want Food", row1[7])
	}

	// Verify second expense (uber ride)
	row2 := records[2]
	if row2[3] != "uber ride" {
		t.Errorf("Row2[Description] = %v, want uber ride", row2[3])
	}
	if row2[4] != "-30.00" {
		t.Errorf("Row2[Amount] = %v, want -30.00", row2[4])
	}
	if row2[6] != "EUR" {
		t.Errorf("Row2[Currency] = %v, want EUR", row2[6])
	}
	if row2[7] != "Transport" {
		t.Errorf("Row2[Category] = %v, want Transport", row2[7])
	}

	// Verify third expense (salary - income type, no category)
	row3 := records[3]
	if row3[3] != "salary" {
		t.Errorf("Row3[Description] = %v, want salary", row3[3])
	}
	if row3[4] != "5000.00" {
		t.Errorf("Row3[Amount] = %v, want 5000.00", row3[4])
	}
	if row3[5] != "income" {
		t.Errorf("Row3[Type] = %v, want income", row3[5])
	}
	if row3[7] != "" {
		t.Errorf("Row3[Category] = %v, want empty string", row3[7])
	}
}

func TestCSVEmpty(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	ctx := context.Background()

	// Get all expenses (should be empty)
	allExpenses, err := s.GetAllExpenseTypes(ctx, user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	// Export to CSV
	var buf bytes.Buffer
	err = CSV(ctx, user.ID(), &buf, allExpenses, s)
	if err != nil {
		t.Fatalf("CSV failed: %v", err)
	}

	// Parse the CSV output
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Verify we only have the header row
	if len(records) != 1 {
		t.Fatalf("Expected 1 row (header only), got %d", len(records))
	}

	expectedHeader := []string{"ID", "Source", "Date", "Description", "Amount", "Type", "Currency", "Category"}
	header := records[0]
	for i, col := range expectedHeader {
		if header[i] != col {
			t.Errorf("Header[%d] = %v, want %v", i, header[i], col)
		}
	}
}

func TestExpenseToCSVRecord(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	ctx := context.Background()
	categoryID, err := s.CreateCategory(ctx, user.ID(), "Shopping", "amazon|store")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	tests := []struct {
		name         string
		expense      storage.Expense
		expectedRow  []string
		validateFunc func(t *testing.T, row []string)
	}{
		{
			name: "Charge with category",
			expense: storage.NewExpense(
				123,
				"Bank",
				"coffee shop",
				"USD",
				-450,
				time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
				storage.ChargeType,
				&categoryID,
			),
			validateFunc: func(t *testing.T, row []string) {
				if row[0] != "123" {
					t.Errorf("ID = %v, want 123", row[0])
				}
				if row[1] != "Bank" {
					t.Errorf("Source = %v, want Bank", row[1])
				}
				if row[2] != "2024-03-10" {
					t.Errorf("Date = %v, want 2024-03-10", row[2])
				}
				if row[3] != "coffee shop" {
					t.Errorf("Description = %v, want coffee shop", row[3])
				}
				if row[4] != "-4.50" {
					t.Errorf("Amount = %v, want -4.50", row[4])
				}
				if row[5] != "charge" {
					t.Errorf("Type = %v, want charge", row[5])
				}
				if row[6] != "USD" {
					t.Errorf("Currency = %v, want USD", row[6])
				}
				if row[7] != "Shopping" {
					t.Errorf("Category = %v, want Shopping", row[7])
				}
			},
		},
		{
			name: "Income without category",
			expense: storage.NewExpense(
				456,
				"Employer",
				"monthly salary",
				"EUR",
				250000,
				time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
				storage.IncomeType,
				nil,
			),
			validateFunc: func(t *testing.T, row []string) {
				if row[0] != "456" {
					t.Errorf("ID = %v, want 456", row[0])
				}
				if row[3] != "monthly salary" {
					t.Errorf("Description = %v, want monthly salary", row[3])
				}
				if row[4] != "2500.00" {
					t.Errorf("Amount = %v, want 2500.00", row[4])
				}
				if row[5] != "income" {
					t.Errorf("Type = %v, want income", row[5])
				}
				if row[7] != "" {
					t.Errorf("Category = %v, want empty string", row[7])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := expenseToCSVRecord(ctx, user.ID(), tt.expense, s)

			if len(row) != 8 {
				t.Fatalf("Expected 8 columns, got %d", len(row))
			}

			tt.validateFunc(t, row)
		})
	}
}

func TestCSVRoundTrip(t *testing.T) {
	// This test verifies that we can export expenses and the format is compatible
	// with potential future import functionality
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	ctx := context.Background()
	categoryID, err := s.CreateCategory(ctx, user.ID(), "Groceries", "supermarket|grocery")
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	// Create a test expense
	testDate := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)
	expenses := []storage.Expense{
		storage.NewExpense(
			0,
			"MyBank",
			"supermarket purchase",
			"GBP",
			-12550,
			testDate,
			storage.ChargeType,
			&categoryID,
		),
	}

	_, err = s.InsertExpenses(ctx, user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert expenses: %v", err)
	}

	// Export to CSV
	allExpenses, err := s.GetAllExpenseTypes(ctx, user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}

	var buf bytes.Buffer
	err = CSV(ctx, user.ID(), &buf, allExpenses, s)
	if err != nil {
		t.Fatalf("CSV failed: %v", err)
	}

	// Verify the CSV is well-formed and parseable
	csvContent := buf.String()

	// Should have header + 1 data row
	lines := strings.Split(strings.TrimSpace(csvContent), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines (header + data), got %d", len(lines))
	}

	// Parse using standard CSV reader
	reader := csv.NewReader(strings.NewReader(csvContent))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse exported CSV: %v", err)
	}

	// Verify data integrity
	dataRow := records[1]
	if dataRow[1] != "MyBank" {
		t.Errorf("Source mismatch: got %v, want MyBank", dataRow[1])
	}
	if dataRow[2] != "2024-06-15" {
		t.Errorf("Date mismatch: got %v, want 2024-06-15", dataRow[2])
	}
	if dataRow[3] != "supermarket purchase" {
		t.Errorf("Description mismatch: got %v, want supermarket purchase", dataRow[3])
	}
	if dataRow[4] != "-125.50" {
		t.Errorf("Amount mismatch: got %v, want -125.50", dataRow[4])
	}
	if dataRow[6] != "GBP" {
		t.Errorf("Currency mismatch: got %v, want GBP", dataRow[6])
	}
	if dataRow[7] != "Groceries" {
		t.Errorf("Category mismatch: got %v, want Groceries", dataRow[7])
	}
}
