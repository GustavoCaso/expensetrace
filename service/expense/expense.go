package expense

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/logger"
	"github.com/GustavoCaso/expensetrace/storage"
)

type Service struct {
	storage storage.Storage
	logger  *logger.Logger
}

func New(storage storage.Storage, logger *logger.Logger) *Service {
	return &Service{
		storage: storage,
		logger:  logger,
	}
}

// List returns the user's expenses matching the given filter and sort options.
func (s *Service) List(
	ctx context.Context,
	userID int64,
	f *domain.ExpenseFilter,
	sortOpts *domain.SortOptions,
) ([]domain.Expense, error) {
	expenses, err := s.storage.GetExpensesFiltered(ctx, userID, f, sortOpts)
	if err != nil {
		s.logger.Error(fmt.Sprintf("error GetExpensesFiltered %s", err.Error()))
		return nil, err
	}
	return expenses, nil
}

// GroupByYearAndMonth groups expenses by year and month, enriching each with
// its category.
func (s *Service) GroupByYearAndMonth(
	ctx context.Context,
	userID int64,
	expenses []domain.Expense,
) (domain.ExpensesByYear, []int, error) {
	groupedExpenses := domain.ExpensesByYear{}
	years := []int{}

	for _, exp := range expenses {
		var category domain.Category

		if exp.CategoryID() != nil {
			c, categoryErr := s.storage.GetCategory(ctx, userID, *exp.CategoryID())
			if categoryErr != nil {
				if !errors.Is(categoryErr, &domain.NotFoundError{}) {
					s.logger.Error(fmt.Sprintf("error GetCategory %s", categoryErr.Error()))
					return groupedExpenses, years, categoryErr
				}
			}
			category = c
		}

		expenseYear := exp.Date().Year()
		expenseMonth := exp.Date().Month().String()

		year, okYear := groupedExpenses[expenseYear]

		if okYear {
			month, okMonth := year[expenseMonth]
			if okMonth {
				expenseview := &domain.ExpenseView{
					Expense: exp,
					Cat:     category,
				}
				month = append(month, expenseview)
			} else {
				expenseview := &domain.ExpenseView{
					Expense: exp,
					Cat:     category,
				}
				month = []*domain.ExpenseView{expenseview}
			}

			year[expenseMonth] = month
		} else {
			years = append(years, expenseYear)
			newYear := map[string][]*domain.ExpenseView{}
			expenseview := &domain.ExpenseView{
				Expense: exp,
				Cat:     category,
			}
			newYear[expenseMonth] = []*domain.ExpenseView{expenseview}
			groupedExpenses[expenseYear] = newYear
		}
	}

	sort.SliceStable(years, func(i, j int) bool {
		return years[i] > years[j]
	})

	return groupedExpenses, years, nil
}

// Get fetches an expense along with its category (a zero-value
// domain.Category if the expense has no category set), returned as a
// domain.ExpenseView ready for rendering.
func (s *Service) Get(ctx context.Context, userID, id int64) (*domain.ExpenseView, error) {
	exp, err := s.storage.GetExpenseByID(ctx, userID, id)
	if err != nil {
		s.logger.Error(fmt.Sprintf("error GetExpenseByID %s", err.Error()))
		return nil, err
	}

	category := domain.EmptyCategory()
	if exp.CategoryID() != nil {
		cat, categoryErr := s.storage.GetCategory(ctx, userID, *exp.CategoryID())
		if categoryErr != nil {
			s.logger.Error(fmt.Sprintf("error GetCategory %s", categoryErr.Error()))
			return nil, categoryErr
		}
		category = cat
	}

	return &domain.ExpenseView{
		Expense: exp,
		Cat:     category,
	}, nil
}

// Create inserts a new expense.
func (s *Service) Create(ctx context.Context, userID int64, e domain.Expense) (domain.Expense, error) {
	created, err := s.storage.InsertExpenses(ctx, userID, []domain.Expense{e})
	if err != nil {
		s.logger.Error(fmt.Sprintf("error InsertExpenses %s", err.Error()))
		return nil, err
	}

	if created != 1 {
		s.logger.Error("error InsertExpenses expense not created")
		return nil, errors.New("expense not created")
	}

	return e, nil
}

// Update updates an expense's fields.
func (s *Service) Update(ctx context.Context, userID int64, e domain.Expense) (int64, error) {
	updated, err := s.storage.UpdateExpense(ctx, userID, e)
	if err != nil {
		s.logger.Error(fmt.Sprintf("error UpdateExpense %s", err.Error()))
		return 0, err
	}
	return updated, nil
}

// Delete deletes an expense.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	_, err := s.storage.DeleteExpense(ctx, userID, id)
	if err != nil {
		s.logger.Error(fmt.Sprintf("error DeleteExpense %s", err.Error()))
		return err
	}
	return nil
}

// Export writes all of the user's expenses as CSV to w.
func (s *Service) Export(ctx context.Context, userID int64, w io.Writer) error {
	expenses, err := s.storage.GetAllExpenseTypes(ctx, userID)
	if err != nil {
		s.logger.Error(fmt.Sprintf("error GetAllExpenseTypes %s", err.Error()))
		return err
	}

	if exportErr := csvExport(ctx, userID, w, expenses, s.storage); exportErr != nil {
		s.logger.Error(fmt.Sprintf("error export.CSV %s", exportErr.Error()))
		return exportErr
	}

	return nil
}
