package testutil

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/storage/sqlite"
)

func SetupTestStorage(t *testing.T, logger *logger.Logger) storage.Storage {
	t.Helper()
	// We use a tempDir + the unique test name (t.Name) that way we can warranty that any test has its own DB
	// Using a tempDir ensure it gets clean after each test
	sqlFile := filepath.Join(t.TempDir(), fmt.Sprintf(":memory:%s", strings.ReplaceAll(t.Name(), "/", ":")))
	stor, err := sqlite.New(sqlFile)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}

	err = stor.ApplyMigrations(logger)
	if err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	t.Cleanup(func() {
		if err = stor.Close(); err != nil {
			t.Errorf("Failed to close test storage: %v", err)
		}
	})

	return stor
}
