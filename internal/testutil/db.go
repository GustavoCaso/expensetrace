package testutil

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/config"
	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/storage/sqlite"
)

func SetupTestStorage(t *testing.T, logger *logger.Logger) storage.Storage {
	t.Helper()
	// We use a tempDir + the unique test name (t.Name) that way we can warranty that any test has its own DB
	// Using a tempDir ensure it gets clean after each test
	sqlFile := filepath.Join(t.TempDir(), fmt.Sprintf(":memory:%s", strings.ReplaceAll(t.Name(), "/", ":")))
	dbConfig := config.DBConfig{
		Source: sqlFile,
	}
	stor, err := sqlite.New(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}

	err = stor.ApplyMigrations(context.Background(), logger)
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
