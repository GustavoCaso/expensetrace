package report

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/testutil"
)

func TestGenerate_BuildsMonthlyReportsBackToFirstExpense(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	twoMonthsAgo := now.AddDate(0, -2, 0)

	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "current month expense", "USD", -1000, now, domain.ChargeType, nil),
		domain.NewExpense(0, "Test Source", "old expense", "USD", -2000, twoMonthsAgo, domain.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)

	svc.Generate(context.Background(), user.ID())

	reports, found := svc.reportsPerUser[user.ID()]
	if !found {
		t.Fatal("Expected reports to be generated for user")
	}

	currentKey := fmt.Sprintf("%d-%d", now.Year(), int(now.Month()))
	oldKey := fmt.Sprintf("%d-%d", twoMonthsAgo.Year(), int(twoMonthsAgo.Month()))

	if _, ok := reports[currentKey]; !ok {
		t.Errorf("Expected report for current month (%s), got keys: %v", currentKey, slices.Collect(maps.Keys(reports)))
	}

	if _, ok := reports[oldKey]; !ok {
		t.Errorf(
			"Expected report for oldest expense month (%s), got keys: %v",
			oldKey,
			slices.Collect(maps.Keys(reports)),
		)
	}

	// Span is 3 calendar months: old month, middle month, current month.
	expectedMonths := 3
	if len(reports) != expectedMonths {
		t.Errorf("Expected %d monthly reports, got %d (%v)",
			expectedMonths, len(reports), slices.Collect(maps.Keys(reports)))
	}
}

func TestForMonth_ReturnsCachedReport(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()

	expenses := []domain.Expense{
		domain.NewExpense(0, "Test Source", "current month expense", "USD", -1000, now, domain.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)
	svc.Generate(context.Background(), user.ID())

	rep := svc.ForMonth(user.ID(), int(now.Month()), now.Year())

	if rep.Spending != -1000 {
		t.Errorf("Expected spending -1000, got %d", rep.Spending)
	}
}

func TestRange_CombinesMonthsAndSummarizesEach(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	day := func(month time.Month, d int) time.Time {
		return time.Date(2024, month, d, 12, 0, 0, 0, time.Local)
	}

	expenses := []domain.Expense{
		domain.NewExpense(0, "src", "salary", "EUR", 300000, day(time.January, 1), domain.IncomeType, nil),
		domain.NewExpense(0, "src", "rent", "EUR", -100000, day(time.January, 2), domain.ChargeType, nil),
		// February has no transactions.
		domain.NewExpense(0, "src", "salary", "EUR", 300000, day(time.March, 1), domain.IncomeType, nil),
		domain.NewExpense(0, "src", "rent", "EUR", -250000, day(time.March, 2), domain.ChargeType, nil),
		// Outside the range; must be ignored.
		domain.NewExpense(0, "src", "rent", "EUR", -999999, day(time.April, 2), domain.ChargeType, nil),
	}
	if _, err := s.InsertExpenses(context.Background(), user.ID(), expenses); err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)

	// from/to are passed reversed on purpose: Range must swap them.
	sel, err := svc.Range(context.Background(), user.ID(), day(time.March, 15), day(time.January, 15))
	if err != nil {
		t.Fatalf("Range: %v", err)
	}

	if sel.Title != "Jan – Mar 2024" {
		t.Errorf("Title = %q, want %q", sel.Title, "Jan – Mar 2024")
	}
	if sel.MonthCount != 3 {
		t.Errorf("MonthCount = %d, want 3", sel.MonthCount)
	}
	if sel.Income != 600000 || sel.Spending != -350000 {
		t.Errorf("Income/Spending = %d/%d, want 600000/-350000", sel.Income, sel.Spending)
	}

	if len(sel.Months) != 3 {
		t.Fatalf("len(Months) = %d, want 3", len(sel.Months))
	}
	if sel.Months[1].HasData {
		t.Errorf("February should have no data: %+v", sel.Months[1])
	}
	if sel.Months[0].Savings != 200000 || sel.Months[2].Savings != 50000 {
		t.Errorf("monthly savings = %d, %d; want 200000, 50000", sel.Months[0].Savings, sel.Months[2].Savings)
	}

	if !sel.IsBest(sel.Months[0]) {
		t.Errorf("Best = %+v, want January", sel.Best)
	}
	if !sel.IsWorst(sel.Months[2]) {
		t.Errorf("Worst = %+v, want March", sel.Worst)
	}
}

func TestRange_NoBestOrWorstWithSingleMonthOfData(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	jan := time.Date(2024, time.January, 10, 12, 0, 0, 0, time.Local)
	expenses := []domain.Expense{
		domain.NewExpense(0, "src", "rent", "EUR", -1000, jan, domain.ChargeType, nil),
	}
	if _, err := s.InsertExpenses(context.Background(), user.ID(), expenses); err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	sel, err := New(s, logger).Range(context.Background(), user.ID(), jan, jan.AddDate(0, 2, 0))
	if err != nil {
		t.Fatalf("Range: %v", err)
	}

	if sel.Best != nil || sel.Worst != nil {
		t.Errorf("expected no best/worst, got %+v / %+v", sel.Best, sel.Worst)
	}
}

func TestYears_NewestFirstIncludingCurrent(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	twoYearsAgo := now.AddDate(-2, 0, 0)
	expenses := []domain.Expense{
		domain.NewExpense(0, "src", "old", "EUR", -1000, twoYearsAgo, domain.ChargeType, nil),
	}
	if _, err := s.InsertExpenses(context.Background(), user.ID(), expenses); err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)
	svc.Generate(context.Background(), user.ID())

	got := svc.Years(user.ID())
	want := []int{now.Year(), now.Year() - 1, now.Year() - 2}
	if !slices.Equal(got, want) {
		t.Errorf("Years() = %v, want %v", got, want)
	}
}

func TestRange_ClampsToFirstTransactionAndCurrentMonth(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	firstExpense := time.Date(now.Year()-1, time.June, 10, 12, 0, 0, 0, time.Local)
	expenses := []domain.Expense{
		domain.NewExpense(0, "src", "first", "EUR", -1000, firstExpense, domain.ChargeType, nil),
		domain.NewExpense(0, "src", "latest", "EUR", -1000, now, domain.ChargeType, nil),
	}
	if _, err := s.InsertExpenses(context.Background(), user.ID(), expenses); err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)

	tests := []struct {
		name                string
		from, to            time.Time
		wantFirst, wantLast string
	}{
		{
			name:      "starts at first transaction",
			from:      time.Date(now.Year()-1, time.January, 1, 0, 0, 0, 0, time.Local),
			to:        time.Date(now.Year()-1, time.December, 1, 0, 0, 0, 0, time.Local),
			wantFirst: fmt.Sprintf("June %d", now.Year()-1),
			wantLast:  fmt.Sprintf("December %d", now.Year()-1),
		},
		{
			name:      "ends at current month",
			from:      time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.Local),
			to:        time.Date(now.Year()+1, time.June, 1, 0, 0, 0, 0, time.Local),
			wantFirst: fmt.Sprintf("January %d", now.Year()),
			wantLast:  fmt.Sprintf("%s %d", now.Month(), now.Year()),
		},
		{
			name:      "range without overlapping data is left unchanged",
			from:      time.Date(now.Year()-3, time.January, 1, 0, 0, 0, 0, time.Local),
			to:        time.Date(now.Year()-3, time.March, 1, 0, 0, 0, 0, time.Local),
			wantFirst: fmt.Sprintf("January %d", now.Year()-3),
			wantLast:  fmt.Sprintf("March %d", now.Year()-3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sel, err := svc.Range(context.Background(), user.ID(), tt.from, tt.to)
			if err != nil {
				t.Fatalf("Range: %v", err)
			}
			if len(sel.Months) == 0 {
				t.Fatal("expected at least one month")
			}

			first, last := sel.Months[0].Title, sel.Months[len(sel.Months)-1].Title
			if first != tt.wantFirst || last != tt.wantLast {
				t.Errorf("months = %s..%s, want %s..%s", first, last, tt.wantFirst, tt.wantLast)
			}
			if sel.MonthCount != len(sel.Months) {
				t.Errorf(
					"MonthCount = %d, want %d (averages must use the clamped range)",
					sel.MonthCount,
					len(sel.Months),
				)
			}
		})
	}
}

func TestRange_UserWithoutExpenses(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	from := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.Local)

	sel, err := New(s, logger).Range(context.Background(), user.ID(), from, now)
	if err != nil {
		t.Fatalf("Range with no expenses should not fail: %v", err)
	}
	if len(sel.Months) != int(now.Month()) {
		t.Errorf("len(Months) = %d, want %d", len(sel.Months), int(now.Month()))
	}
}
