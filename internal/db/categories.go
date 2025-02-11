package db

import (
	"database/sql"
	"log"

	"github.com/GustavoCaso/expensetrace/internal/config"
)

var createCategoriesTableStatement = `
CREATE TABLE IF NOT EXISTS categories
(
 id INTEGER PRIMARY KEY,
 name TEXT NOT NULL,
 pattern TEXT NOT NULL,
 UNIQUE(name) ON CONFLICT FAIL
) STRICT;
`

type Category struct {
	ID      int
	Name    string
	Pattern string
	Total   int
}

func CreateCategoriesTable(db *sql.DB) error {
	// Create table
	statement, err := db.Prepare(createCategoriesTableStatement)
	if err != nil {
		return err
	}

	_, err = statement.Exec()

	return err
}

func PopulateCategoriesFromConfig(db *sql.DB, conf *config.Config) []error {
	// Insert records
	insertStmt, err := db.Prepare("INSERT INTO categories(name, pattern) values(?, ?)")

	errors := []error{}

	if err != nil {
		errors = append(errors, err)
		return errors
	}
	for _, category := range conf.Categories {
		_, err := insertStmt.Exec(category.Name, category.Pattern)
		if err != nil {
			errors = append(errors, ErrInsert{
				err: err,
			})
		}
	}

	return errors
}

func GetCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.Query("SELECT * FROM categories")
	if err != nil {
		return []Category{}, err
	}

	defer rows.Close()

	categories := []Category{}

	for rows.Next() {
		var category Category
		var id int

		if err := rows.Scan(&id, &category.Name, &category.Pattern); err != nil {
			log.Fatal(err)
		}

		category.ID = id

		categories = append(categories, category)
	}

	return categories, nil
}

func GetCategory(db *sql.DB, categoryID int) (Category, error) {
	row := db.QueryRow("SELECT * FROM categories WHERE id=$1", categoryID)
	var id int
	var name string
	var pattern string
	err := row.Scan(&id, &name, &pattern)

	if err != nil {
		return Category{}, err
	}

	return Category{
		ID:      id,
		Name:    name,
		Pattern: pattern,
	}, nil
}

func GetCategoriesAndTotalExpenses(db *sql.DB) ([]Category, error) {
	rows, err := db.Query("SELECT c.name , c.id, COUNT(e.category_id) FROM categories AS c JOIN expenses AS e ON c.id = e.category_id GROUP BY c.name ORDER BY c.id;")
	if err != nil {
		return []Category{}, err
	}

	defer rows.Close()

	categories := []Category{}

	for rows.Next() {
		var category Category
		var id int
		var total int

		if err := rows.Scan(&category.Name, &id, &total); err != nil {
			log.Fatal(err)
		}

		category.ID = id
		category.Total = total

		categories = append(categories, category)
	}

	return categories, nil
}

func DeleteCategoriesDB(db *sql.DB) error {
	// drop table
	statement, err := db.Prepare("DROP TABLE IF EXISTS categories;")
	if err != nil {
		return err
	}

	_, err = statement.Exec()

	if err != nil {
		return err
	}

	return nil
}
