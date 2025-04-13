package cli

import (
	"database/sql"
	"flag"
	"fmt"
	"testing"

	"github.com/GustavoCaso/expensetrace/internal/category"
)

// mockCommand implements the Command interface for testing.
type mockCommand struct {
	description string
	runError    error
}

func (c mockCommand) SetFlags(fset *flag.FlagSet) {
	fset.String("test", "", "test flag")
}

func (c mockCommand) Description() string {
	return c.description
}

func (c mockCommand) Run(_ *sql.DB, _ *category.Matcher) error {
	return c.runError
}

func TestCommandInterface(t *testing.T) {
	// Test successful command
	cmd := mockCommand{
		description: "Test command",
		runError:    nil,
	}

	// Test SetFlags
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(fs)
	if fs.Lookup("test") == nil {
		t.Error("SetFlags() did not register the test flag")
	}

	// Test Description
	desc := cmd.Description()
	if desc != "Test command" {
		t.Errorf("Description() = %v, want %v", desc, "Test command")
	}

	// Test Run
	err := cmd.Run(nil, nil)
	if err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}

	// Test command with error
	cmdWithError := mockCommand{
		description: "Error command",
		runError:    fmt.Errorf("test error"),
	}

	err = cmdWithError.Run(nil, nil)
	if err == nil {
		t.Error("Run() expected error, got nil")
	}
	if err.Error() != "test error" {
		t.Errorf("Run() error = %v, want %v", err, "test error")
	}
}
