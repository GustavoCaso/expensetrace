package cli

import (
	"database/sql"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/logger"
)

type Command interface {
	Description() string
	Run(db *sql.DB, matcher *category.Matcher, logger *logger.Logger) error
}
