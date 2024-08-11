package command

import (
	"flag"

	"github.com/GustavoCaso/expensetrace/pkg/config"
)

type Command interface {
	SetFlags(fset *flag.FlagSet)
	Run(conf *config.Config)
}
