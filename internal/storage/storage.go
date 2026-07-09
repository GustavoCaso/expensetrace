package storage

import (
	"context"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/domain"
	"github.com/GustavoCaso/expensetrace/internal/logger"
)

type Storage interface {
	// Migrations
	ApplyMigrations(ctx context.Context, logger *logger.Logger) error

	// Users
	CreateUser(ctx context.Context, username, passwordHash string) (domain.User, error)
	GetUserByUsername(ctx context.Context, username string) (domain.User, error)
	GetUserByID(ctx context.Context, id int64) (domain.User, error)
	UpdateUsername(ctx context.Context, userID int64, newUsername string) error
	UpdatePassword(ctx context.Context, userID int64, newPasswordHash string) error

	// Sessions
	CreateSession(ctx context.Context, userID int64, sessionID string, expiresAt time.Time) (domain.Session, error)
	GetSession(ctx context.Context, sessionID string) (domain.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteExpiredSessions(ctx context.Context) error

	// Expenses
	GetExpenseByID(ctx context.Context, userID, id int64) (domain.Expense, error)
	UpdateExpense(ctx context.Context, userID int64, expense domain.Expense) (int64, error)
	DeleteExpense(ctx context.Context, userID, id int64) (int64, error)
	InsertExpenses(ctx context.Context, userID int64, expenses []domain.Expense) (int64, error)
	GetExpenses(ctx context.Context, userID int64) ([]domain.Expense, error)
	GetAllExpenseTypes(ctx context.Context, userID int64) ([]domain.Expense, error)
	UpdateExpenses(ctx context.Context, userID int64, expenses []domain.Expense) (int64, error)
	GetExpensesFromDateRange(
		ctx context.Context,
		userID int64,
		start time.Time,
		end time.Time,
	) ([]domain.Expense, error)
	GetExpensesWithoutCategory(ctx context.Context, userID int64) ([]domain.Expense, error)
	GetExpensesWithoutCategoryWithQuery(ctx context.Context, userID int64, keyword string) ([]domain.Expense, error)
	SearchExpenses(ctx context.Context, userID int64, keyword string) ([]domain.Expense, error)
	SearchExpensesByDescription(ctx context.Context, userID int64, description string) ([]domain.Expense, error)
	GetFirstExpense(ctx context.Context, userID int64) (domain.Expense, error)
	GetExpensesByCategory(ctx context.Context, userID, categoryID int64) ([]domain.Expense, error)
	GetExpensesFiltered(
		ctx context.Context,
		userID int64,
		expFilter *domain.ExpenseFilter,
		sort *domain.SortOptions,
	) ([]domain.Expense, error)

	// Categories
	GetCategories(ctx context.Context, userID int64) ([]domain.Category, error)
	GetCategory(ctx context.Context, userID, categoryID int64) (domain.Category, error)
	DeleteCategory(ctx context.Context, userID, categoryID int64) (int64, error)
	UpdateCategory(ctx context.Context, userID, categoryID int64, name, pattern string, monthlyBudget int64) error
	CreateCategory(ctx context.Context, userID int64, name, pattern string, monthlyBudget int64) (int64, error)
	DeleteCategories(ctx context.Context, userID int64) (int64, error)
	GetExcludeCategory(ctx context.Context, userID int64) (domain.Category, error)

	// Resource managment
	Close() error
}
