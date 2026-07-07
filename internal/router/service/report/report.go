package report

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/domain"
	"github.com/GustavoCaso/expensetrace/internal/logger"
	pkgreport "github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

type Service struct {
	storage        storage.Storage
	logger         *logger.Logger
	reportsPerUser map[int64]map[string]pkgreport.Report
}

func New(storage storage.Storage, logger *logger.Logger) *Service {
	return &Service{
		storage:        storage,
		logger:         logger,
		reportsPerUser: map[int64]map[string]pkgreport.Report{},
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

	reports := map[string]pkgreport.Report{}

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

		result, reportErr := pkgreport.Generate(ctx, userID, firstDay, lastDay, s.storage, expenses, "monthly")

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

// ChartData returns the user's cached reports as chart data points, ordered
// oldest-to-newest.
func (s *Service) ChartData(userID int64) []domain.ChartDataPoint {
	reports := s.reportsPerUser[userID]
	reportKeys := slices.Collect(maps.Keys(reports))

	sort.SliceStable(reportKeys, func(i, j int) bool {
		s1 := strings.Split(reportKeys[i], "-")
		s2 := strings.Split(reportKeys[j], "-")
		year1, _ := strconv.Atoi(s1[0])
		month1, _ := strconv.Atoi(s1[1])

		year2, _ := strconv.Atoi(s2[0])
		month2, _ := strconv.Atoi(s2[1])

		if year1 == year2 {
			return time.Month(month1) > time.Month(month2)
		}

		return year1 > year2
	})

	chartData := make([]domain.ChartDataPoint, 0, len(reports))

	for _, key := range reportKeys {
		parts := strings.Split(key, "-")
		rep := reports[key]

		chartData = append(chartData, domain.ChartDataPoint{
			Month:             rep.Title,
			Income:            rep.Income,
			URL:               fmt.Sprintf("/?month=%s&year=%s", parts[1], parts[0]),
			Spending:          rep.Spending,
			Savings:           rep.Savings,
			SavingsPercentage: rep.SavingsPercentage,
		})
	}

	// Reverse the order to have oldest months first (better for chart visualization)
	for i, j := 0, len(chartData)-1; i < j; i, j = i+1, j-1 {
		chartData[i], chartData[j] = chartData[j], chartData[i]
	}

	return chartData
}

// ForMonth returns the cached report for the given month/year, or a
// zero-value report if none has been generated.
func (s *Service) ForMonth(userID int64, month, year int) pkgreport.Report {
	return s.reportsPerUser[userID][fmt.Sprintf("%d-%d", year, month)]
}
