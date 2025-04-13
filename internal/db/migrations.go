package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

func createMigrationsTable(db *sql.DB) error {
	statement, err := db.Prepare(`
			CREATE TABLE IF NOT EXISTS schema_migrations (
					version INTEGER PRIMARY KEY,
					applied_at INTEGER NOT NULL
			)
	`)
	if err != nil {
		return err
	}
	_, err = statement.Exec()
	return err
}

func DropTables(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for dropping tables: %w", err)
	}

	// drop tables
	_, err = tx.Exec("DROP TABLE IF EXISTS expenses;")
	if err != nil {
		rErr := tx.Rollback()
		if rErr != nil {
			return rErr
		}
		return err
	}

	_, err = tx.Exec("DROP TABLE IF EXISTS categories;")
	if err != nil {
		rErr := tx.Rollback()
		if rErr != nil {
			return rErr
		}
		return err
	}

	_, err = tx.Exec("DROP TABLE IF EXISTS schema_migrations;")
	if err != nil {
		rErr := tx.Rollback()
		if rErr != nil {
			return rErr
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit deletion: %w", err)
	}

	return nil
}

func ApplyMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current schema version
	currentVersion := 0
	row := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations")
	if err := row.Scan(&currentVersion); err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	// Define migrations
	migrations := []struct {
		version int
		up      func(*sql.Tx) error
	}{
		{
			version: 1,
			up: func(tx *sql.Tx) error {
				// Create Expense Table
				_, err := tx.Exec(`
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
			version: 2,
			up: func(tx *sql.Tx) error {
				// Create Categories Table
				_, err := tx.Exec(`
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
			version: 3,
			up: func(tx *sql.Tx) error {
				// Set all category_id from 0 to NULL
				// That way we can run the next migration
				_, err := tx.Exec(`
				UPDATE expenses 
				SET category_id = NULL 
				WHERE category_id = 0;
				`)
				return err
			},
		},
		{
			version: 4,
			up: func(tx *sql.Tx) error {
				// Add foreign key constraint to expenses table
				_, err := tx.Exec(`
				PRAGMA foreign_keys=OFF;
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
				INSERT INTO expenses_new SELECT * FROM expenses;
				DROP TABLE expenses;
				ALTER TABLE expenses_new RENAME TO expenses;
				PRAGMA foreign_key_check;
				`)
				return err
			},
		},
		// Add more migrations as your schema evolves
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if migration.version > currentVersion {
			log.Printf("Applying migration %d...", migration.version)

			// Begin transaction for this migration
			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin transaction for migration %d: %w",
					migration.version, err)
			}

			// Apply migration
			if err := migration.up(tx); err != nil {
				rErr := tx.Rollback()
				if rErr != nil {
					return rErr
				}
				return fmt.Errorf("migration %d failed: %w", migration.version, err)
			}

			// Record migration
			_, err = tx.Exec(
				"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
				migration.version, time.Now().Unix(),
			)
			if err != nil {
				rErr := tx.Rollback()
				if rErr != nil {
					return rErr
				}
				return fmt.Errorf("failed to record migration %d: %w",
					migration.version, err)
			}

			// Commit transaction
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit migration %d: %w",
					migration.version, err)
			}

			log.Printf("Migration %d applied successfully", migration.version)
		}
	}

	return nil
}
