package cli

import (
	"database/sql"
	"flag"

	"github.com/GustavoCaso/expensetrace/internal/config"
)

type Command interface {
	SetFlags(fset *flag.FlagSet)
	Description() string
	Run(conf *config.Config, db *sql.DB)
}
