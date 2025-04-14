package router

import (
	"database/sql"

	"github.com/GustavoCaso/expensetrace/internal/category"
)

// newTestRouter creates a router instance for testing.
func newTestRouter(db *sql.DB, matcher *category.Matcher) *router {
	return &router{
		reload:  false,
		matcher: matcher,
		db:      db,
	}
}
