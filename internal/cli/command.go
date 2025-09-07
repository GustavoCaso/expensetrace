package cli

import (
	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/storage"
)

type Command interface {
	Description() string
	Run(storage storage.Storage, matcher *category.Matcher, logger *logger.Logger) error
}
