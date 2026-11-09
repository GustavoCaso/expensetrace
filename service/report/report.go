package report

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/logger"
	"github.com/GustavoCaso/expensetrace/storage"
	"github.com/GustavoCaso/expensetrace/util"
)

type Service struct {
	storage        storage.Storage
	logger         *logger.Logger
	reportsPerUser map[int64]map[string]domain.Report
}

func New(storage storage.Storage, logger *logger.Logger) *Service {
	return &Service{
		storage:        storage,
		logger:         logger,
		reportsPerUser: map[int64]map[string]domain.Report{},
	}
}

// Generate builds monthly reports for the user, walking backwards from the
// current month to the month of the user's first expense, and caches them.
func (s *Service) Generate(ctx context.Context, userID int64) {
	now := time.Now()
	month := now.Month()
	year := now.Year()
	skipYear := false
	ex, err := s.storage.GetFirstExpense(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to generate reports", "error", err, "userID", userID)
		return
	}

	lastMonth := ex.Date().Month()
	lastYear := ex.Date().Year()

	reports := map[string]domain.Report{}

	for year >= lastYear {
		if month == time.January {
			skipYear = true
		}

		firstDay, lastDay := util.GetMonthDates(int(month), year)

		expenses, expenseErr := s.storage.GetExpensesFromDateRange(ctx, userID, firstDay, lastDay)

		if expenseErr != nil {
			s.logger.Warn("Failed to generate reports", "error", expenseErr, "userID", userID)
			return
		}

		result, reportErr := generate(ctx, userID, firstDay, lastDay, s.storage, expenses)

		if reportErr != nil {
			s.logger.Warn("Failed to generate reports", "error", reportErr, "userID", userID)
			return
		}

		reports[fmt.Sprintf("%d-%d", year, month)] = result

		if skipYear {
			year--
			month = time.December
			skipYear = false
			continue
		}

		if year == lastYear && month == lastMonth {
			break
		}

		month--
	}

	s.reportsPerUser[userID] = reports
}

// ForMonth returns the cached report for the given month/year, or a
// zero-value report if none has been generated.
func (s *Service) ForMonth(userID int64, month, year int) domain.Report {
	return s.reportsPerUser[userID][fmt.Sprintf("%d-%d", year, month)]
}

// Month builds the report for the given month/year directly from storage,
// bypassing the cache used by the web UI. Callers outside the browser flow
// (e.g. the MCP server) use it so they never see a stale or empty report.
func (s *Service) Month(ctx context.Context, userID int64, month, year int) (domain.Report, error) {
	firstDay, lastDay := util.GetMonthDates(month, year)

	expenses, err := s.storage.GetExpensesFromDateRange(ctx, userID, firstDay, lastDay)
	if err != nil {
		return domain.Report{}, err
	}

	return generate(ctx, userID, firstDay, lastDay, s.storage, expenses)
}

// Range builds a report covering every month from the month of `from` to
// the month of `to`, inclusive, plus a summary of each month in between.
// Like Month, it reads from storage rather than the UI cache. The arguments
// are swapped if `to` is before `from`.
//
// The range is clamped to months that can hold data: it starts no earlier
// than the user's first transaction and ends no later than the current
// month, so empty months don't drag down averages or inflate budgets.
func (s *Service) Range(ctx context.Context, userID int64, from, to time.Time) (domain.SelectionReport, error) {
	if to.Before(from) {
		from, to = to, from
	}

	from, to, err := s.clampToData(ctx, userID, from, to)
	if err != nil {
		return domain.SelectionReport{}, err
	}

	firstDay, _ := util.GetMonthDates(int(from.Month()), from.Year())
	_, lastDay := util.GetMonthDates(int(to.Month()), to.Year())

	expenses, err := s.storage.GetExpensesFromDateRange(ctx, userID, firstDay, lastDay)
	if err != nil {
		return domain.SelectionReport{}, err
	}

	total, err := generate(ctx, userID, firstDay, lastDay, s.storage, expenses)
	if err != nil {
		return domain.SelectionReport{}, err
	}

	// Bucket the already-loaded expenses by month instead of querying again.
	type monthKey struct {
		year  int
		month time.Month
	}
	byMonth := map[monthKey][]domain.Expense{}
	for _, ex := range expenses {
		d := ex.Date().In(firstDay.Location())
		key := monthKey{d.Year(), d.Month()}
		byMonth[key] = append(byMonth[key], ex)
	}

	selection := domain.SelectionReport{Report: total}
	for monthStart := firstDay; !monthStart.After(lastDay); monthStart = monthStart.AddDate(0, 1, 0) {
		start, end := util.GetMonthDates(int(monthStart.Month()), monthStart.Year())
		monthExpenses := byMonth[monthKey{start.Year(), start.Month()}]

		rep, monthErr := generate(ctx, userID, start, end, s.storage, monthExpenses)
		if monthErr != nil {
			return domain.SelectionReport{}, monthErr
		}

		selection.Months = append(selection.Months, domain.MonthSummary{
			Title:             rep.Title,
			Month:             int(start.Month()),
			Year:              start.Year(),
			Income:            rep.Income,
			Spending:          rep.Spending,
			Savings:           rep.Savings,
			SavingsPercentage: rep.SavingsPercentage,
			HasData:           rep.Income != 0 || rep.Spending != 0,
		})
	}

	selection.Best, selection.Worst = bestAndWorstMonths(selection.Months)

	return selection, nil
}

// clampToData narrows [from, to] to the months between the user's first
// transaction and the current month. If the range doesn't overlap that span
// at all (e.g. a year before any data) it is returned unchanged, so the
// report shows the requested months as empty rather than failing.
func (s *Service) clampToData(
	ctx context.Context,
	userID int64,
	from, to time.Time,
) (time.Time, time.Time, error) {
	start, end := monthStart(from), monthStart(to)

	if current := monthStart(time.Now()); end.After(current) {
		end = current
	}

	first, err := s.storage.GetFirstExpense(ctx, userID)
	var notFoundErr *domain.NotFoundError
	switch {
	case errors.As(err, &notFoundErr):
		// No transactions yet; nothing to clamp the start to.
	case err != nil:
		return from, to, err
	default:
		if firstMonth := monthStart(first.Date()); start.Before(firstMonth) {
			start = firstMonth
		}
	}

	if end.Before(start) {
		return from, to, nil
	}

	return start, end, nil
}

// monthStart returns midnight on the first day of t's month, in local time.
func monthStart(t time.Time) time.Time {
	t = t.In(time.Local)
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
}

// bestAndWorstMonths returns the months with the highest and lowest savings,
// ignoring months without data. Both are nil when fewer than two months have
// data, since a single month is trivially both.
func bestAndWorstMonths(months []domain.MonthSummary) (*domain.MonthSummary, *domain.MonthSummary) {
	var best, worst *domain.MonthSummary
	withData := 0

	for i := range months {
		m := &months[i]
		if !m.HasData {
			continue
		}
		withData++
		if best == nil || m.Savings > best.Savings {
			best = m
		}
		if worst == nil || m.Savings < worst.Savings {
			worst = m
		}
	}

	if withData < 2 { //nolint:mnd // need two months to compare
		return nil, nil
	}

	return best, worst
}

// Years returns the years covered by the user's cached reports, newest
// first, always including the current year. Call Generate first.
func (s *Service) Years(userID int64) []int {
	currentYear := time.Now().Year()
	seen := map[int]bool{currentYear: true}

	for key := range s.reportsPerUser[userID] {
		yearPart, _, _ := strings.Cut(key, "-")
		if year, err := strconv.Atoi(yearPart); err == nil {
			seen[year] = true
		}
	}

	years := slices.Collect(maps.Keys(seen))
	slices.Sort(years)
	slices.Reverse(years)

	return years
}

// CategoryBudget pairs a budgeted category with its status for a month.
type CategoryBudget struct {
	Name   string
	Budget domain.BudgetInfo
}

// Budgets returns the budget status for every category with a monthly
// budget set, for the given month/year. Unlike the report, it includes
// budgeted categories that had no spending that month.
func (s *Service) Budgets(ctx context.Context, userID int64, month, year int) ([]CategoryBudget, error) {
	rep, err := s.Month(ctx, userID, month, year)
	if err != nil {
		return nil, err
	}

	categories, err := s.storage.GetCategories(ctx, userID)
	if err != nil {
		return nil, err
	}

	spentByName := make(map[string]int64, len(rep.ExpenseCategories))
	for _, c := range rep.ExpenseCategories {
		spentByName[c.Name] = c.Amount
	}

	budgets := []CategoryBudget{}
	for _, c := range categories {
		if c.MonthlyBudget() <= 0 || c.Name() == domain.ExcludeCategory {
			continue
		}
		budgets = append(budgets, CategoryBudget{
			Name:   c.Name(),
			Budget: calculateBudgetInfo(c.MonthlyBudget(), spentByName[c.Name()]),
		})
	}

	sort.SliceStable(budgets, func(i, j int) bool {
		return budgets[i].Budget.PercentageUsed > budgets[j].Budget.PercentageUsed
	})

	return budgets, nil
}
