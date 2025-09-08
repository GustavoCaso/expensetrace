package cli

import (
	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

type Command interface {
	Description() string
	Run(storage storage.Storage, matcher *matcher.Matcher, logger *logger.Logger) error
}
