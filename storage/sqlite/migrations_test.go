package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/GustavoCaso/expensetrace/domain"
	"github.com/GustavoCaso/expensetrace/logger"
)

func TestMigrations(t *testing.T) {
	stor, user := setupTestStorage(t)

	// Test that storage was created successfully with migrations applied
	// This verifies that all tables exist and are properly structured
	_, err := stor.GetExpenses(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to query expenses table after migrations: %v", err)
	}

	_, err = stor.GetCategories(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("Failed to query categories table after migrations: %v", err)
	}

	// Test that we can create data (which verifies schema)
	_, err = stor.CreateCategory(context.Background(), user.ID(), "Test Category", "test.*", 0, domain.ChargeCategory)
	if err != nil {
		t.Fatalf("Failed to create category after migrations: %v", err)
	}
}

func TestCategoryTypeMigration(t *testing.T) {
	stor, user := setupTestStorage(t)
	ctx := context.Background()

	foodID, err := stor.CreateCategory(ctx, user.ID(), "Food", "food", 0, domain.ChargeCategory)
	if err != nil {
		t.Fatalf("Failed to create category: %v", err)
	}

	exclude, err := stor.GetExcludeCategory(ctx, user.ID())
	if err != nil {
		t.Fatalf("Failed to get exclude category: %v", err)
	}
	excludeID := exclude.ID()

	now := time.Now()
	_, err = stor.InsertExpenses(ctx, user.ID(), []domain.Expense{
		domain.NewExpense(0, "bank", "food charge", "EUR", -1000, now, domain.ChargeType, &foodID),
		domain.NewExpense(0, "bank", "food refund", "EUR", 500, now, domain.IncomeType, &foodID),
		domain.NewExpense(0, "bank", "own transfer", "EUR", 9000, now, domain.IncomeType, &excludeID),
	})
	if err != nil {
		t.Fatalf("Failed to insert expenses: %v", err)
	}

	// Roll the schema back to before the migration, as an existing database
	// would be, then migrate again.
	db := stor.(*sqliteStorage).db
	for _, stmt := range []string{
		"DELETE FROM schema_migrations WHERE version = (SELECT MAX(version) FROM schema_migrations)",
		"ALTER TABLE categories DROP COLUMN type",
	} {
		if _, err = db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("Failed to roll back migration (%s): %v", stmt, err)
		}
	}

	if err = stor.ApplyMigrations(ctx, logger.New(logger.Config{})); err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	food, err := stor.GetCategory(ctx, user.ID(), foodID)
	if err != nil {
		t.Fatalf("Failed to get category: %v", err)
	}
	if food.Type() != domain.ChargeCategory {
		t.Errorf("Food type = %v, want charge", food.Type())
	}

	exclude, err = stor.GetExcludeCategory(ctx, user.ID())
	if err != nil {
		t.Fatalf("Failed to get exclude category: %v", err)
	}
	if exclude.Type() != domain.AnyCategory {
		t.Errorf("Exclude type = %v, want any", exclude.Type())
	}

	expenses, err := stor.GetAllExpenseTypes(ctx, user.ID())
	if err != nil {
		t.Fatalf("Failed to get expenses: %v", err)
	}
	if len(expenses) != 3 {
		t.Fatalf("Expected 3 expenses, got %d", len(expenses))
	}

	want := map[string]*int64{
		"food charge":  &foodID,
		"food refund":  nil, // income unlinked from charge category
		"own transfer": &excludeID,
	}
	for _, ex := range expenses {
		wantID, ok := want[ex.Description()]
		if !ok {
			continue
		}
		got := ex.CategoryID()
		switch {
		case wantID == nil && got != nil:
			t.Errorf("%s: category = %d, want none", ex.Description(), *got)
		case wantID != nil && (got == nil || *got != *wantID):
			t.Errorf("%s: category = %v, want %d", ex.Description(), got, *wantID)
		}
	}
}
