package report

import (
	"context"

	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/testutil"
)

func TestGenerate(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	// Create test expenses
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, startDate, domain.ChargeType, nil),
		domain.NewExpense(
			0,
			"Test Source",
			"Uber ride",
			"USD",
			-50000,
			startDate.Add(24*time.Hour),
			domain.ChargeType,
			nil,
		),
		domain.NewExpense(
			0,
			"Test Source",
			"Salary",
			"USD",
			5000000,
			startDate.Add(48*time.Hour),
			domain.IncomeType,
			nil,
		),
	}

	// Test monthly report
	report, err := generate(context.Background(), user.ID(), startDate, endDate, s, expenses)

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

	expectedEarningsPerDay := int64(5000000 / 31)
	if report.EarningsPerDay != expectedEarningsPerDay {
		t.Errorf("Report.EarningsPerDay = %v, want %v", report.EarningsPerDay, expectedEarningsPerDay)
	}

	expectedSpendingPerDay := int64(173456 / 31)
	if report.AverageSpendingPerDay != expectedSpendingPerDay {
		t.Errorf("Report.AverageSpendingPerDay = %v, want %v", report.AverageSpendingPerDay, expectedSpendingPerDay)
	}

	// A full calendar year is titled by its year and covers 12 months.
	endOfYear := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	yearlyReport, err := generate(context.Background(), user.ID(), startDate, endOfYear, s, expenses)
	if err != nil {
		t.Fatalf("Got error generating report: %s", err.Error())
	}

	if yearlyReport.Title != "2024" {
		t.Errorf("Report.Title = %v, want 2024", yearlyReport.Title)
	}

	if yearlyReport.MonthCount != 12 {
		t.Errorf("Report.MonthCount = %d, want 12", yearlyReport.MonthCount)
	}
}

func TestReportTitle(t *testing.T) {
	date := func(year int, month time.Month) time.Time {
		return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	}

	tests := []struct {
		name       string
		start, end time.Time
		want       string
	}{
		{"single month", date(2026, time.November), date(2026, time.November), "November 2026"},
		{"full year", date(2026, time.January), date(2026, time.December), "2026"},
		{"within a year", date(2026, time.January), date(2026, time.November), "Jan – Nov 2026"},
		{"across years", date(2025, time.March), date(2026, time.February), "Mar 2025 – Feb 2026"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reportTitle(tt.start, tt.end); got != tt.want {
				t.Errorf("reportTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerate_ScalesBudgetByMonths(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	// 100.00 monthly budget; 250.00 spent over three months is under the
	// 300.00 range budget, but would be "over" against a single month.
	catID, err := s.CreateCategory(context.Background(), user.ID(), "Food", "food", 10000, domain.ChargeCategory)
	if err != nil {
		t.Fatalf("Error creating category: %s", err.Error())
	}

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC)
	expenses := []domain.Expense{
		domain.NewExpense(0, "src", "food", "EUR", -25000, start, domain.ChargeType, &catID),
	}

	rep, err := generate(context.Background(), user.ID(), start, end, s, expenses)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(rep.ExpenseCategories) != 1 {
		t.Fatalf("expected 1 expense category, got %d", len(rep.ExpenseCategories))
	}

	budget := rep.ExpenseCategories[0].Budget
	if budget.Amount != 30000 {
		t.Errorf("Budget.Amount = %d, want 30000", budget.Amount)
	}
	if budget.Status != domain.BudgetStatusNear {
		t.Errorf("Budget.Status = %q, want %q", budget.Status, domain.BudgetStatusNear)
	}
}

func TestCategories(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	catID, catErr := s.CreateCategory(
		context.Background(),
		user.ID(),
		"Food",
		"restaurant|food|grocery",
		0,
		domain.ChargeCategory,
	)
	if catErr != nil {
		t.Fatalf("Error creating category: %s", catErr.Error())
	}

	// Test with duplicate expenses
	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "Restaurant bill", "USD", -123456, startDate, domain.ChargeType, nil),
		domain.NewExpense(
			0,
			"Test Source",
			"Restaurant bill",
			"USD",
			-123456,
			startDate.Add(24*time.Hour),
			domain.ChargeType,
			nil,
		), // Duplicate description
		domain.NewExpense(
			0,
			"Test Source",
			"Salary",
			"USD",
			5000000,
			startDate.Add(48*time.Hour),
			domain.IncomeType,
			nil,
		),
		domain.NewExpense(
			0,
			"Test Source",
			"Expense with category",
			"USD",
			-123456,
			startDate.Add(48*time.Hour),
			domain.ChargeType,
			&catID,
		),
		domain.NewExpense(
			0,
			"Refund from restaurant",
			"Income with same category as an expense",
			"USD",
			678,
			startDate.Add(48*time.Hour),
			domain.IncomeType,
			&catID,
		),
	}

	expenseCategories, incomeCategories, totalIncome, totalSpending, err := splitByExpenseType(
		context.Background(),
		user.ID(),
		s,
		expenses,
		1,
	)

	if err != nil {
		t.Fatalf("Got error generating categories: %s", err.Error())
	}

	// Verify income and spending
	if totalIncome != 5000678 {
		t.Errorf("income = %v, want 5000678", totalIncome)
	}
	if totalSpending != -370368 {
		t.Errorf("spending = %v, want -370368", totalSpending)
	}

	// Verify categories
	// uncategorized charge
	// Food
	// income

	if len(incomeCategories) != 1 {
		t.Fatalf("income categories must be one. got: %d", len(incomeCategories))
	}

	for _, expense := range incomeCategories[0].Expenses {
		if expense.Type() != domain.IncomeType {
			t.Fatal("income category includes expense")
		}
	}

	if len(expenseCategories) != 2 {
		t.Fatalf("income categories must be two. got: %d", len(expenseCategories))
	}

	// Categories are ordered by amount, largest first
	if expenseCategories[0].Name != "uncategorized charge" || expenseCategories[1].Name != "Food" {
		t.Fatalf(
			"expense categories must be ordered by amount. got: %s, %s",
			expenseCategories[0].Name,
			expenseCategories[1].Name,
		)
	}

	validExpenseCategories := map[string]struct{}{
		"uncategorized charge": {},
		"Food":                 {},
	}

	for _, expenseCategory := range expenseCategories {
		_, ok := validExpenseCategories[expenseCategory.Name]
		if !ok {
			t.Fatalf("unexpected expense category %s", expenseCategory.Name)
		}

		for _, expense := range expenseCategory.Expenses {
			if expense.Type() != domain.ChargeType {
				t.Fatal("uncategorized charge category includes income")
			}
		}
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

func TestGenerate_PercentageOfIncome(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	tests := []struct {
		name     string
		expenses []domain.Expense
		want     float64
	}{
		{
			name: "share of income",
			expenses: []domain.Expense{
				domain.NewExpense(0, "src", "salary", "EUR", 400000, start, domain.IncomeType, nil),
				domain.NewExpense(0, "src", "rent", "EUR", -100000, start, domain.ChargeType, nil),
			},
			want: 25,
		},
		{
			name: "can exceed 100 when spending outruns income",
			expenses: []domain.Expense{
				domain.NewExpense(0, "src", "salary", "EUR", 100000, start, domain.IncomeType, nil),
				domain.NewExpense(0, "src", "rent", "EUR", -150000, start, domain.ChargeType, nil),
			},
			want: 150,
		},
		{
			name: "zero without income",
			expenses: []domain.Expense{
				domain.NewExpense(0, "src", "rent", "EUR", -100000, start, domain.ChargeType, nil),
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep, err := generate(context.Background(), user.ID(), start, end, s, tt.expenses)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			if len(rep.ExpenseCategories) != 1 {
				t.Fatalf("expected 1 expense category, got %d", len(rep.ExpenseCategories))
			}

			got := rep.ExpenseCategories[0].PercentageOfIncome
			if got != tt.want {
				t.Errorf("PercentageOfIncome = %v, want %v", got, tt.want)
			}
			if bar := rep.ExpenseCategories[0].IncomeBarWidth(); bar > 100 {
				t.Errorf("IncomeBarWidth = %v, must be capped at 100", bar)
			}
		})
	}
}

func TestIncomeCategories(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)
	ctx := context.Background()

	salaryID, err := s.CreateCategory(ctx, user.ID(), "Salary", "payroll", 0, domain.IncomeCategory)
	if err != nil {
		t.Fatalf("Error creating category: %s", err.Error())
	}

	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	expenses := []domain.Expense{
		domain.NewExpense(0, "bank", "payroll jan", "EUR", 300000, date, domain.IncomeType, &salaryID),
		domain.NewExpense(0, "bank", "payroll bonus", "EUR", 50000, date, domain.IncomeType, &salaryID),
		domain.NewExpense(0, "bank", "gift", "EUR", 1000, date, domain.IncomeType, nil),
	}

	_, incomeCategories, totalIncome, _, err := splitByExpenseType(ctx, user.ID(), s, expenses, 1)
	if err != nil {
		t.Fatalf("splitByExpenseType returned error: %s", err.Error())
	}

	if totalIncome != 351000 {
		t.Errorf("income = %d, want 351000", totalIncome)
	}

	got := map[string]int64{}
	for _, c := range incomeCategories {
		got[c.Name] = c.Amount
		if c.Budget.Status != domain.BudgetStatusNoBudget {
			t.Errorf("%s: budget status = %s, want no budget", c.Name, c.Budget.Status)
		}
	}

	want := map[string]int64{"Salary": 350000, "uncategorized income": 1000}
	if len(got) != len(want) {
		t.Fatalf("income categories = %v, want %v", got, want)
	}
	for name, amount := range want {
		if got[name] != amount {
			t.Errorf("%s = %d, want %d", name, got[name], amount)
		}
	}

	// Largest first.
	if incomeCategories[0].Name != "Salary" {
		t.Errorf("first income category = %s, want Salary", incomeCategories[0].Name)
	}
}
