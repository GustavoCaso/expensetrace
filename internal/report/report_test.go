package report

import (
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestGenerate(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)
	// Create test expenses
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, startDate, storage.ChargeType, nil),
		storage.NewExpense(
			0,
			"Test Source",
			"Uber ride",
			"USD",
			-50000,
			startDate.Add(24*time.Hour),
			storage.ChargeType,
			nil,
		),
		storage.NewExpense(
			0,
			"Test Source",
			"Salary",
			"USD",
			5000000,
			startDate.Add(48*time.Hour),
			storage.IncomeType,
			nil,
		),
	}

	// Test monthly report
	report, err := Generate(startDate, endDate, s, expenses, "monthly")

	if err != nil {
		t.Fatalf("Got error generating report: %s", err.Error())
	}

	// Verify report fields
	if report.Title != "January 2024" {
		t.Errorf("Report.Title = %v, want January 2024", report.Title)
	}

	if report.Spending != -173456 {
		t.Errorf("Report.Spending = %v, want -173456", report.Spending)
	}

	if report.Income != 5000000 {
		t.Errorf("Report.Income = %v, want 5000000", report.Income)
	}

	expectedSavings := int64(5000000 - 173456)
	if report.Savings != expectedSavings {
		t.Errorf("Report.Savings = %v, want %v", report.Savings, expectedSavings)
	}

	expectedSavingsPercentage := (float32(expectedSavings) / float32(5000000)) * 100
	if report.SavingsPercentage != expectedSavingsPercentage {
		t.Errorf("Report.SavingsPercentage = %v, want %v", report.SavingsPercentage, expectedSavingsPercentage)
	}

	expectedEarningsPerDay := int64(5000000 / 30)
	if report.EarningsPerDay != expectedEarningsPerDay {
		t.Errorf("Report.EarningsPerDay = %v, want %v", report.EarningsPerDay, expectedEarningsPerDay)
	}

	expectedSpendingPerDay := int64(173456 / 30)
	if report.AverageSpendingPerDay != expectedSpendingPerDay {
		t.Errorf("Report.AverageSpendingPerDay = %v, want %v", report.AverageSpendingPerDay, expectedSpendingPerDay)
	}

	// Test yearly report
	yearlyReport, err := Generate(startDate, endDate, s, expenses, "yearly")
	if err != nil {
		t.Fatalf("Got error generating report: %s", err.Error())
	}

	if yearlyReport.Title != "2024" {
		t.Errorf("Report.Title = %v, want 2024", yearlyReport.Title)
	}
}

func TestCategories(t *testing.T) {
	logger := testutil.TestLogger(t)
	s := testutil.SetupTestStorage(t, logger)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	catID, catErr := s.CreateCategory("Food", "restaurant|food|grocery")
	if catErr != nil {
		t.Fatalf("Error creating category: %s", catErr.Error())
	}

	// Test with duplicate expenses
	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, startDate, storage.ChargeType, nil),
		storage.NewExpense(
			0,
			"Test Source",
			"Restaurant bill",
			"USD",
			-123456,
			startDate.Add(24*time.Hour),
			storage.ChargeType,
			nil,
		), // Duplicate description
		storage.NewExpense(
			0,
			"Test Source",
			"Salary",
			"USD",
			5000000,
			startDate.Add(48*time.Hour),
			storage.IncomeType,
			nil,
		),
		storage.NewExpense(
			0,
			"Test Source",
			"Expense with category",
			"USD",
			-123456,
			startDate.Add(48*time.Hour),
			storage.ChargeType,
			&catID,
		),
	}

	categories, duplicates, income, spending, err := Categories(s, expenses)

	if err != nil {
		t.Fatalf("Got error generating categories: %s", err.Error())
	}

	// Verify duplicates
	if len(duplicates) != 1 {
		t.Errorf("len(duplicates) = %v, want 1", len(duplicates))
	}
	if duplicates[0] != "Restaurant bill" {
		t.Errorf("duplicates[0] = %v, want Restaurant bill", duplicates[0])
	}

	// Verify income and spending
	if income != 5000000 {
		t.Errorf("income = %v, want 5000000", income)
	}
	if spending != -370368 {
		t.Errorf("spending = %v, want -370368", spending)
	}

	// Verify categories
	// uncategorized charge
	// Food
	// income
	if len(categories) != 3 {
		t.Errorf("len(categories) = %v, want 3", len(categories))
	}
}

func TestCalendarDays(t *testing.T) {
	testCases := []struct {
		name     string
		t1       time.Time
		t2       time.Time
		expected int
	}{
		{
			name:     "Same day",
			t1:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			t2:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "One day difference",
			t1:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			t2:       time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "Month difference",
			t1:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			t2:       time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			expected: 31,
		},
		{
			name:     "Year difference",
			t1:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			t2:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: 366,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			days := calendarDays(tc.t1, tc.t2)
			if days != tc.expected {
				t.Errorf("calendarDays(%v, %v) = %v, want %v", tc.t1, tc.t2, days, tc.expected)
			}
		})
	}
}
