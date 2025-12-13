package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/logger"
)

type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "record not found"
}

type Category interface {
	ID() int64
	Name() string
	Pattern() string
	MonthlyBudget() int64
}

type category struct {
	id            int64
	name          string
	pattern       string
	monthlyBudget int64
}

func (c category) ID() int64 {
	return c.id
}

func (c category) Name() string {
	return c.name
}

func (c category) Pattern() string {
	return c.pattern
}

func (c category) MonthlyBudget() int64 {
	return c.monthlyBudget
}

func NewCategory(id int64, name, pattern string, monthlyBudget int64) Category {
	return category{
		id:            id,
		name:          name,
		pattern:       pattern,
		monthlyBudget: monthlyBudget,
	}
}

func EmptyCategory() Category {
	return NewCategory(0, "", "", 0)
}

type ExpenseType int

const (
	ChargeType ExpenseType = iota
	IncomeType
)

type Expense interface {
	ID() int64
	Source() string
	Date() time.Time
	Description() string
	Amount() int64
	Type() ExpenseType
	Currency() string
	CategoryID() *int64
}

type expense struct {
	id          int64
	source      string
	date        time.Time
	description string
	amount      int64
	expenseType ExpenseType
	currency    string
	categoryID  *int64
}

func NewExpense(
	id int64,
	source, description, currency string,
	amount int64,
	date time.Time,
	expenseType ExpenseType,
	categoryID *int64,
) Expense {
	return &expense{
		id:          id,
		source:      source,
		description: description,
		amount:      amount,
		date:        date,
		expenseType: expenseType,
		currency:    currency,
		categoryID:  categoryID,
	}
}

func (e *expense) ID() int64 {
	return e.id
}

func (e *expense) Source() string {
	return e.source
}

func (e *expense) Date() time.Time {
	return e.date
}

func (e *expense) Description() string {
	return e.description
}

func (e *expense) Amount() int64 {
	return e.amount
}

func (e *expense) Type() ExpenseType {
	return e.expenseType
}

func (e *expense) Currency() string {
	return e.currency
}

func (e *expense) CategoryID() *int64 {
	return e.categoryID
}

func (e *expense) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"id":           e.id,
		"source":       e.source,
		"date":         e.date,
		"description":  e.description,
		"amount":       e.amount,
		"expense_type": e.expenseType,
		"currency":     e.currency,
		"category_id":  e.categoryID,
	})
}

const ExcludeCategory = "🚫 Exclude"

type User interface {
	ID() int64
	Username() string
	PasswordHash() string
	CreatedAt() time.Time
}

type user struct {
	id           int64
	username     string
	passwordHash string
	createdAt    time.Time
}

func (u *user) ID() int64 {
	return u.id
}

func (u *user) Username() string {
	return u.username
}

func (u *user) PasswordHash() string {
	return u.passwordHash
}

func (u *user) CreatedAt() time.Time {
	return u.createdAt
}

func NewUser(id int64, username, passwordHash string, createdAt time.Time) User {
	return &user{
		id:           id,
		username:     username,
		passwordHash: passwordHash,
		createdAt:    createdAt,
	}
}

type Session interface {
	ID() string
	UserID() int64
	ExpiresAt() time.Time
	CreatedAt() time.Time
}

type session struct {
	id        string
	userID    int64
	expiresAt time.Time
	createdAt time.Time
}

func (s *session) ID() string {
	return s.id
}

func (s *session) UserID() int64 {
	return s.userID
}

func (s *session) ExpiresAt() time.Time {
	return s.expiresAt
}

func (s *session) CreatedAt() time.Time {
	return s.createdAt
}

func NewSession(id string, userID int64, expiresAt, createdAt time.Time) Session {
	return &session{
		id:        id,
		userID:    userID,
		expiresAt: expiresAt,
		createdAt: createdAt,
	}
}

type Storage interface {
	// Migrations
	ApplyMigrations(ctx context.Context, logger *logger.Logger) error

	// Users
	CreateUser(ctx context.Context, username, passwordHash string) (User, error)
	GetUserByUsername(ctx context.Context, username string) (User, error)
	GetUserByID(ctx context.Context, id int64) (User, error)
	UpdateUsername(ctx context.Context, userID int64, newUsername string) error
	UpdatePassword(ctx context.Context, userID int64, newPasswordHash string) error

	// Sessions
	CreateSession(ctx context.Context, userID int64, sessionID string, expiresAt time.Time) (Session, error)
	GetSession(ctx context.Context, sessionID string) (Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteExpiredSessions(ctx context.Context) error

	// Expenses
	GetExpenseByID(ctx context.Context, userID, id int64) (Expense, error)
	UpdateExpense(ctx context.Context, userID int64, expense Expense) (int64, error)
	DeleteExpense(ctx context.Context, userID, id int64) (int64, error)
	InsertExpenses(ctx context.Context, userID int64, expenses []Expense) (int64, error)
	GetExpenses(ctx context.Context, userID int64) ([]Expense, error)
	GetAllExpenseTypes(ctx context.Context, userID int64) ([]Expense, error)
	UpdateExpenses(ctx context.Context, userID int64, expenses []Expense) (int64, error)
	GetExpensesFromDateRange(ctx context.Context, userID int64, start time.Time, end time.Time) ([]Expense, error)
	GetExpensesWithoutCategory(ctx context.Context, userID int64) ([]Expense, error)
	GetExpensesWithoutCategoryWithQuery(ctx context.Context, userID int64, keyword string) ([]Expense, error)
	SearchExpenses(ctx context.Context, userID int64, keyword string) ([]Expense, error)
	SearchExpensesByDescription(ctx context.Context, userID int64, description string) ([]Expense, error)
	GetFirstExpense(ctx context.Context, userID int64) (Expense, error)
	GetExpensesByCategory(ctx context.Context, userID, categoryID int64) ([]Expense, error)

	// Categories
	GetCategories(ctx context.Context, userID int64) ([]Category, error)
	GetCategory(ctx context.Context, userID, categoryID int64) (Category, error)
	DeleteCategory(ctx context.Context, userID, categoryID int64) (int64, error)
	UpdateCategory(ctx context.Context, userID, categoryID int64, name, pattern string, monthlyBudget int64) error
	CreateCategory(ctx context.Context, userID int64, name, pattern string, monthlyBudget int64) (int64, error)
	DeleteCategories(ctx context.Context, userID int64) (int64, error)
	GetExcludeCategory(ctx context.Context, userID int64) (Category, error)

	// Resource managment
	Close() error
}
