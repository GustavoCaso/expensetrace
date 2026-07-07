package report

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestGenerate_BuildsMonthlyReportsBackToFirstExpense(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	twoMonthsAgo := now.AddDate(0, -2, 0)

	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "current month expense", "USD", -1000, now, storage.ChargeType, nil),
		storage.NewExpense(0, "Test Source", "old expense", "USD", -2000, twoMonthsAgo, storage.ChargeType, nil),
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

func TestChartData_OrdersOldestFirst(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()
	twoMonthsAgo := now.AddDate(0, -2, 0)

	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "current month expense", "USD", -1000, now, storage.ChargeType, nil),
		storage.NewExpense(0, "Test Source", "old expense", "USD", -2000, twoMonthsAgo, storage.ChargeType, nil),
	}

	_, err := s.InsertExpenses(context.Background(), user.ID(), expenses)
	if err != nil {
		t.Fatalf("Failed to insert test expenses: %v", err)
	}

	svc := New(s, logger)
	svc.Generate(context.Background(), user.ID())

	chartData := svc.ChartData(user.ID())

	if len(chartData) != 3 {
		t.Fatalf("Expected 3 chart points, got %d", len(chartData))
	}

	expectedFirst := twoMonthsAgo.Month().String()
	expectedLast := now.Month().String()

	if !strings.HasPrefix(chartData[0].Month, expectedFirst) {
		t.Errorf("Expected first chart point to be oldest month %q, got %q", expectedFirst, chartData[0].Month)
	}

	if !strings.HasPrefix(chartData[len(chartData)-1].Month, expectedLast) {
		t.Errorf(
			"Expected last chart point to be newest month %q, got %q",
			expectedLast,
			chartData[len(chartData)-1].Month,
		)
	}
}

func TestForMonth_ReturnsCachedReport(t *testing.T) {
	logger := testutil.TestLogger(t)
	s, user := testutil.SetupTestStorage(t, logger)

	now := time.Now()

	expenses := []storage.Expense{
		storage.NewExpense(0, "Test Source", "current month expense", "USD", -1000, now, storage.ChargeType, nil),
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
