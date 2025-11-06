package router

import (
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/testutil"
)

func TestNew(t *testing.T) {
	logger := testutil.TestLogger(t)
	database, _ := testutil.SetupTestStorage(t, logger)

	// Create router
	handler, _ := New(database, logger)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
}
