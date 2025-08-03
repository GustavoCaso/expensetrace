package testutil

import (
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/logger"
)

func TestLogger(t *testing.T) *logger.Logger {
	t.Helper()

	// creates a test logger that doesn't output anything.
	testLogger := logger.New(logger.Config{
		Level:  logger.LevelInfo,
		Format: logger.FormatText,
		Output: "discard",
	})

	return testLogger
}
