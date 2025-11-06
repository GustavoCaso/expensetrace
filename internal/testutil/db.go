package testutil

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/storage/sqlite"
)

func SetupTestStorage(t *testing.T, logger *logger.Logger) (storage.Storage, storage.User) {
	t.Helper()
	// We use a tempDir + the unique test name (t.Name) that way we can warranty that any test has its own DB
	// Using a tempDir ensure it gets clean after each test
	sqlFile := filepath.Join(t.TempDir(), fmt.Sprintf(":memory:%s", strings.ReplaceAll(t.Name(), "/", ":")))
	stor, err := sqlite.New(sqlFile)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}

	err = stor.ApplyMigrations(t.Context(), logger)
	if err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	t.Cleanup(func() {
		if err = stor.Close(); err != nil {
			t.Errorf("Failed to close test storage: %v", err)
		}
	})

	hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte("test"), bcrypt.DefaultCost)
	if hashErr != nil {
		t.Fatalf("error creating password hash: %v", hashErr)
	}

	user, userErr := stor.CreateUser(t.Context(), "test", string(hashedPassword))
	if userErr != nil {
		t.Fatalf("error creating user: %v", userErr)
	}

	return stor, user
}
