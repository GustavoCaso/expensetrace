package cli

import (
	"database/sql"
	"flag"

	"github.com/GustavoCaso/expensetrace/internal/category"
)

type Command interface {
	SetFlags(fset *flag.FlagSet)
	Description() string
	Run(db *sql.DB, matcher *category.Matcher)
}
