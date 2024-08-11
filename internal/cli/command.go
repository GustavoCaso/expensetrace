package cli

import (
	"flag"

	"github.com/GustavoCaso/expensetrace/internal/config"
)

type Command interface {
	SetFlags(fset *flag.FlagSet)
	Run(conf *config.Config)
}
