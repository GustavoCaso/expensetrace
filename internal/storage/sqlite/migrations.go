package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

func createMigrationsTable(db *sql.DB) error {
	ctx := context.Background()

	statement, err := db.PrepareContext(ctx, `
			CREATE TABLE IF NOT EXISTS schema_migrations (
					version INTEGER PRIMARY KEY,
					applied_at INTEGER NOT NULL
			)
	`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.ExecContext(context.Background())
	return err
}

func DropTables(db *sql.DB) error {
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for dropping tables: %w", err)
	}

	// drop tables (in order to respect foreign keys)
	_, err = tx.ExecContext(ctx, "DROP TABLE IF EXISTS expenses;")
	if err != nil {
		rErr := tx.Rollback()
		if rErr != nil {
			return rErr
		}
		return err
	}

	_, err = tx.ExecContext(ctx, "DROP TABLE IF EXISTS categories;")
	if err != nil {
		rErr := tx.Rollback()
		if rErr != nil {
			return rErr
		}
		return err
	}

	_, err = tx.ExecContext(ctx, "DROP TABLE IF EXISTS sessions;")
	if err != nil {
		rErr := tx.Rollback()
		if rErr != nil {
			return rErr
		}
		return err
	}

	_, err = tx.ExecContext(ctx, "DROP TABLE IF EXISTS users;")
	if err != nil {
		rErr := tx.Rollback()
		if rErr != nil {
			return rErr
		}
		return err
	}

	_, err = tx.ExecContext(ctx, "DROP TABLE IF EXISTS schema_migrations;")
	if err != nil {
		rErr := tx.Rollback()
		if rErr != nil {
			return rErr
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit deletion: %w", err)
	}

	return nil
}

func (s *sqliteStorage) ApplyMigrations(ctx context.Context, logger *logger.Logger) error {
	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(s.db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current schema version
	currentVersion := 0
	row := s.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations")
	if err := row.Scan(&currentVersion); err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	// Define migrations
	migrations := []struct {
		name string
		up   func(*sql.Tx) error
	}{
		{
			name: "Create expenses table",
			up: func(tx *sql.Tx) error {
				// Create Expense Table
				_, err := tx.ExecContext(ctx, `
					CREATE TABLE IF NOT EXISTS expenses
					(
					id INTEGER PRIMARY KEY,
					source TEXT,
					amount INTEGER NOT NULL,
					description TEXT NOT NULL,
					expense_type INTEGER NOT NULL,
					date INTEGER NOT NULL,
					currency TEXT NOT NULL,
					category_id INTEGER,
					UNIQUE(source, date, description, amount) ON CONFLICT FAIL
					) STRICT;`)
				return err
			},
		},
		{
			name: "Create categories table",
			up: func(tx *sql.Tx) error {
				// Create Categories Table
				_, err := tx.ExecContext(ctx, `
					CREATE TABLE IF NOT EXISTS categories
					(
					 id INTEGER PRIMARY KEY,
					 name TEXT NOT NULL,
					 pattern TEXT NOT NULL,
					 UNIQUE(name) ON CONFLICT FAIL
					) STRICT;`)
				return err
			},
		},
		{
			name: "Set category_id to NULL",
			up: func(tx *sql.Tx) error {
				// Set all category_id from 0 to NULL
				// That way we can run the next migration
				_, err := tx.ExecContext(ctx, `
				UPDATE expenses 
				SET category_id = NULL 
				WHERE category_id = 0;
				`)
				return err
			},
		},
		{
			name: "Set foreign key constraints expenses <-> categories",
			up: func(tx *sql.Tx) error {
				// Disable foreign keys temporarily
				if _, err := tx.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
					return fmt.Errorf("failed to disable foreign keys: %w", err)
				}

				// Create new table with foreign key constraint
				_, err := tx.ExecContext(ctx, `
					CREATE TABLE expenses_new (
						id INTEGER PRIMARY KEY,
						source TEXT,
						amount INTEGER NOT NULL,
						description TEXT NOT NULL,
						expense_type INTEGER NOT NULL,
						date INTEGER NOT NULL,
						currency TEXT NOT NULL,
						category_id INTEGER,
						UNIQUE(source, date, description, amount) ON CONFLICT FAIL,
						FOREIGN KEY(category_id) REFERENCES categories(id)
					) STRICT;
				`)
				if err != nil {
					return fmt.Errorf("failed to create expenses_new table: %w", err)
				}

				// Copy data from old table
				if _, err = tx.ExecContext(ctx, "INSERT INTO expenses_new SELECT * FROM expenses"); err != nil {
					return fmt.Errorf("failed to copy data to expenses_new: %w", err)
				}

				// Drop old table
				if _, err = tx.ExecContext(ctx, "DROP TABLE expenses"); err != nil {
					return fmt.Errorf("failed to drop old expenses table: %w", err)
				}

				// Rename new table
				if _, err = tx.ExecContext(ctx, "ALTER TABLE expenses_new RENAME TO expenses"); err != nil {
					return fmt.Errorf("failed to rename expenses_new table: %w", err)
				}

				// Re-enable foreign keys
				if _, err = tx.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
					return fmt.Errorf("failed to re-enable foreign keys: %w", err)
				}

				// Check for foreign key violations
				rows, err := tx.QueryContext(ctx, "PRAGMA foreign_key_check(expenses)")
				if err != nil || rows.Err() != nil {
					return fmt.Errorf("failed to check foreign keys: %w", errors.Join(err, rows.Err()))
				}
				defer rows.Close()

				// If there are any violations, fail the migration
				if rows.Next() {
					var table, rowid, parent, fkid string
					if scanErr := rows.Scan(&table, &rowid, &parent, &fkid); scanErr != nil {
						return fmt.Errorf("failed to read foreign key violation: %w", scanErr)
					}
					return fmt.Errorf(
						"foreign key constraint violation: table=%s, rowid=%s, parent=%s, fkid=%s",
						table,
						rowid,
						parent,
						fkid,
					)
				}

				return nil
			},
		},
		{
			name: "Add type column to categories",
			up: func(tx *sql.Tx) error {
				// 1. First add the column with a default value of 0 (expense)
				_, alterErr := tx.ExecContext(ctx, `
						ALTER TABLE categories ADD COLUMN type INTEGER NOT NULL DEFAULT 0;
				`)
				if alterErr != nil {
					return alterErr
				}

				// 2. Fetch all categories to analyze them
				rows, err := tx.QueryContext(ctx, "SELECT id, name, pattern FROM categories")
				if err != nil {
					return err
				}
				defer rows.Close()

				type categoryInfo struct {
					id      int
					name    string
					pattern string
				}

				var categories []categoryInfo
				for rows.Next() {
					var cat categoryInfo
					if scanError := rows.Scan(&cat.id, &cat.name, &cat.pattern); scanError != nil {
						return scanError
					}
					categories = append(categories, cat)
				}
				if rowsErr := rows.Err(); rowsErr != nil {
					return rowsErr
				}

				// 3. For each category, analyze transactions to determine if it's income or expense
				for _, cat := range categories {
					// Analyze the transactions in this category
					expenseTotalRow := tx.QueryRowContext(ctx, `
						SELECT SUM(amount) FROM expenses WHERE category_id = ?
					`, cat.id)
					var totalAmount int64
					if totalErr := expenseTotalRow.Scan(&totalAmount); totalErr != nil {
						return totalErr
					}

					categoryType := 0 // Default to expense
					if totalAmount > 0 {
						categoryType = 1
					}

					_, err = tx.ExecContext(ctx,
						"UPDATE categories SET type = ? WHERE id = ?", categoryType, cat.id)
					if err != nil {
						return err
					}
				}

				return nil
			},
		},
		{
			name: "Remove column from categories",
			up: func(tx *sql.Tx) error {
				// 1. First add the column with a default value of 0 (expense)
				_, alterErr := tx.ExecContext(ctx, `
						ALTER TABLE categories DROP COLUMN type;
				`)
				if alterErr != nil {
					return alterErr
				}

				return nil
			},
		},
		{
			name: "Add exclude category",
			up: func(tx *sql.Tx) error {
				_, alterErr := tx.ExecContext(ctx, `
						INSERT INTO categories (name, pattern) values(?, "$a")
				`, storage.ExcludeCategory)
				if alterErr != nil {
					return alterErr
				}

				return nil
			},
		},
		{
			name: "Create users table",
			up: func(tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
					CREATE TABLE IF NOT EXISTS users (
						id INTEGER PRIMARY KEY,
						username TEXT NOT NULL,
						password_hash TEXT NOT NULL,
						created_at INTEGER NOT NULL,
						UNIQUE(username) ON CONFLICT FAIL
					) STRICT;`)
				if err != nil {
					return err
				}

				// Create default admin user for existing data
				// Password: "admin" (bcrypt hash)
				// Users should change this on first login
				_, err = tx.ExecContext(ctx, `
					INSERT INTO users (id, username, password_hash, created_at)
					VALUES (1, 'admin', '$2a$10$1DMMhCw0qMlNedcIxHpVjeJzGCjIN1JWyR.QLz7YzljbzEj4Jgsem', ?)
				`, time.Now().Unix())
				return err
			},
		},
		{
			name: "Create sessions table",
			up: func(tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
					CREATE TABLE IF NOT EXISTS sessions (
						id TEXT PRIMARY KEY,
						user_id INTEGER NOT NULL,
						expires_at INTEGER NOT NULL,
						created_at INTEGER NOT NULL,
						FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
					) STRICT;`)
				return err
			},
		},
		{
			name: "Add user_id to expenses table",
			up: func(tx *sql.Tx) error {
				// Disable foreign keys temporarily
				if _, err := tx.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
					return fmt.Errorf("failed to disable foreign keys: %w", err)
				}

				// Create new table with user_id column and foreign key constraints
				_, err := tx.ExecContext(ctx, `
					CREATE TABLE expenses_new (
						id INTEGER PRIMARY KEY,
						source TEXT,
						amount INTEGER NOT NULL,
						description TEXT NOT NULL,
						expense_type INTEGER NOT NULL,
						date INTEGER NOT NULL,
						currency TEXT NOT NULL,
						category_id INTEGER,
						user_id INTEGER NOT NULL DEFAULT 1,
						UNIQUE(source, date, description, amount, user_id) ON CONFLICT FAIL,
						FOREIGN KEY(category_id) REFERENCES categories(id),
						FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
					) STRICT;
				`)
				if err != nil {
					return fmt.Errorf("failed to create expenses_new table: %w", err)
				}

				// Copy data from old table with default user_id=1
				_, err = tx.ExecContext(ctx, `
					INSERT INTO expenses_new (id, source, amount, description, expense_type, date, currency, category_id, user_id)
					SELECT id, source, amount, description, expense_type, date, currency, category_id, 1 FROM expenses
				`)
				if err != nil {
					return fmt.Errorf("failed to copy data to expenses_new: %w", err)
				}

				// Drop old table
				if _, err = tx.ExecContext(ctx, "DROP TABLE expenses"); err != nil {
					return fmt.Errorf("failed to drop old expenses table: %w", err)
				}

				// Rename new table
				if _, err = tx.ExecContext(ctx, "ALTER TABLE expenses_new RENAME TO expenses"); err != nil {
					return fmt.Errorf("failed to rename expenses_new table: %w", err)
				}

				// Re-enable foreign keys
				if _, err = tx.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
					return fmt.Errorf("failed to re-enable foreign keys: %w", err)
				}

				// Check for foreign key violations
				rows, err := tx.QueryContext(ctx, "PRAGMA foreign_key_check(expenses)")
				if err != nil || rows.Err() != nil {
					return fmt.Errorf("failed to check foreign keys: %w", errors.Join(err, rows.Err()))
				}
				defer rows.Close()

				// If there are any violations, fail the migration
				if rows.Next() {
					var table, rowid, parent, fkid string
					if scanErr := rows.Scan(&table, &rowid, &parent, &fkid); scanErr != nil {
						return fmt.Errorf("failed to read foreign key violation: %w", scanErr)
					}
					return fmt.Errorf(
						"foreign key constraint violation: table=%s, rowid=%s, parent=%s, fkid=%s",
						table,
						rowid,
						parent,
						fkid,
					)
				}

				return nil
			},
		},
		{
			name: "Add user_id to categories table",
			up: func(tx *sql.Tx) error {
				// Disable foreign keys temporarily
				if _, err := tx.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
					return fmt.Errorf("failed to disable foreign keys: %w", err)
				}

				// Create new categories table with user_id column
				_, err := tx.ExecContext(ctx, `
					CREATE TABLE categories_new (
						id INTEGER PRIMARY KEY,
						name TEXT NOT NULL,
						pattern TEXT NOT NULL,
						user_id INTEGER NOT NULL,
						UNIQUE(name, user_id) ON CONFLICT FAIL,
						FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
					) STRICT;
				`)
				if err != nil {
					return fmt.Errorf("failed to create categories_new table: %w", err)
				}

				// Copy data from old categories table with default user_id=1
				_, err = tx.ExecContext(ctx, `
					INSERT INTO categories_new (id, name, pattern, user_id)
					SELECT id, name, pattern, 1 FROM categories
				`)
				if err != nil {
					return fmt.Errorf("failed to copy data to categories_new: %w", err)
				}

				// Create temporary expenses table (needed because expenses has FK to categories)
				// This is necessary because SQLite won't let us drop categories while expenses references it
				_, err = tx.ExecContext(ctx, `
					CREATE TABLE expenses_temp (
						id INTEGER PRIMARY KEY,
						source TEXT,
						amount INTEGER NOT NULL,
						description TEXT NOT NULL,
						expense_type INTEGER NOT NULL,
						date INTEGER NOT NULL,
						currency TEXT NOT NULL,
						category_id INTEGER,
						user_id INTEGER NOT NULL DEFAULT 1,
						UNIQUE(source, date, description, amount, user_id) ON CONFLICT FAIL,
						FOREIGN KEY(category_id) REFERENCES categories_new(id),
						FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
					) STRICT;
				`)
				if err != nil {
					return fmt.Errorf("failed to create expenses_temp table: %w", err)
				}

				// Copy data from expenses
				_, err = tx.ExecContext(ctx, `
					INSERT INTO expenses_temp SELECT * FROM expenses
				`)
				if err != nil {
					return fmt.Errorf("failed to copy data to expenses_temp: %w", err)
				}

				// Now we can drop both old tables
				if _, err = tx.ExecContext(ctx, "DROP TABLE expenses"); err != nil {
					return fmt.Errorf("failed to drop old expenses table: %w", err)
				}

				if _, err = tx.ExecContext(ctx, "DROP TABLE categories"); err != nil {
					return fmt.Errorf("failed to drop old categories table: %w", err)
				}

				// Rename new tables
				if _, err = tx.ExecContext(ctx, "ALTER TABLE categories_new RENAME TO categories"); err != nil {
					return fmt.Errorf("failed to rename categories_new table: %w", err)
				}

				if _, err = tx.ExecContext(ctx, "ALTER TABLE expenses_temp RENAME TO expenses"); err != nil {
					return fmt.Errorf("failed to rename expenses_temp table: %w", err)
				}

				// Re-enable foreign keys
				if _, err = tx.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
					return fmt.Errorf("failed to re-enable foreign keys: %w", err)
				}

				// Check for foreign key violations in both tables
				rows, err := tx.QueryContext(ctx, "PRAGMA foreign_key_check")
				if err != nil || rows.Err() != nil {
					return fmt.Errorf("failed to check foreign keys: %w", errors.Join(err, rows.Err()))
				}
				defer rows.Close()

				// If there are any violations, fail the migration
				if rows.Next() {
					var table, rowid, parent, fkid string
					if scanErr := rows.Scan(&table, &rowid, &parent, &fkid); scanErr != nil {
						return fmt.Errorf("failed to read foreign key violation: %w", scanErr)
					}
					return fmt.Errorf(
						"foreign key constraint violation: table=%s, rowid=%s, parent=%s, fkid=%s",
						table,
						rowid,
						parent,
						fkid,
					)
				}

				return nil
			},
		},
	}

	// Apply pending migrations
	for i, migration := range migrations {
		// Check if migration is already applied
		migrationVersion := i + 1
		//nolint:nestif // No need to extract this code to a function as is clear
		if migrationVersion > currentVersion {
			logger.Info("Applying migration",
				"version", migrationVersion,
				"name", migration.name)

			// Begin transaction for this migration
			tx, err := s.db.BeginTx(ctx, nil)
			if err != nil {
				return fmt.Errorf("failed to begin transaction for migration %d: %w",
					migrationVersion, err)
			}

			// Apply migration
			if err = migration.up(tx); err != nil {
				rErr := tx.Rollback()
				if rErr != nil {
					return rErr
				}
				return fmt.Errorf("migration %d failed: %w", migrationVersion, err)
			}

			// Record migration
			_, err = tx.ExecContext(ctx,
				"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
				migrationVersion, time.Now().Unix(),
			)
			if err != nil {
				rErr := tx.Rollback()
				if rErr != nil {
					return rErr
				}
				return fmt.Errorf("failed to record migration %d: %w",
					migrationVersion, err)
			}

			// Commit transaction
			if err = tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit migration %d: %w",
					migrationVersion, err)
			}

			logger.Info("Migration applied successfully", "version", migrationVersion)
		}
	}

	return nil
}
