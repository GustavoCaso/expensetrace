package router

import (
	"database/sql"

	"github.com/GustavoCaso/expensetrace/internal/category"
	_ "github.com/mattn/go-sqlite3"
)

// newTestRouter creates a router instance for testing
func newTestRouter(db *sql.DB, matcher *category.Matcher) *router {
	return &router{
		reload:  false,
		matcher: matcher,
		db:      db,
	}
}
