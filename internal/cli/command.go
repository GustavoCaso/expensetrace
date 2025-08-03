package cli

import (
	"database/sql"
	"flag"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/logger"
)

type Command interface {
	SetFlags(fset *flag.FlagSet)
	Description() string
	Run(db *sql.DB, matcher *category.Matcher, logger *logger.Logger) error
}
