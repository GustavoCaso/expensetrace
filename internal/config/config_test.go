package config

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/GustavoCaso/expensetrace/internal/logger"
)

func TestParse(t *testing.T) {
	// Create a temporary test config file
	config := Config{
		DB: "test.db",
		Logger: logger.Config{
			Level:  "info",
			Format: "json",
			Output: "discard",
		},
	}
	content, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	tmpfile, err := os.CreateTemp(t.TempDir(), "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err = tmpfile.Write(content); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	if err = tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temporary file: %v", err)
	}

	// Test parsing the config file
	conf, err := Parse(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify database path
	if conf.DB != "test.db" {
		t.Errorf("Expected DB path 'test.db', got '%s'", conf.DB)
	}

	if conf.Logger.Format != "json" {
		t.Fatalf("Expected logger format 'json', got '%s'", conf.Logger.Format)
	}

	if conf.Logger.Level != "info" {
		t.Fatalf("Expected logger level 'info', got '%s'", conf.Logger.Level)
	}

	if conf.Logger.Output != "discard" {
		t.Fatalf("Expected logger output 'discard', got '%s'", conf.Logger.Output)
	}
}

func TestParseENV(t *testing.T) {
	t.Setenv("EXPENSETRACE_DB", "test.db")
	t.Setenv("EXPENSETRACE_LOG_LEVEL", "info")
	t.Setenv("EXPENSETRACE_LOG_FORMAT", "json")
	t.Setenv("EXPENSETRACE_LOG_OUTPUT", "discard")

	// Test parsing the config file
	conf, err := Parse("noexiting.yml")
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify database path
	if conf.DB != "test.db" {
		t.Errorf("Expected DB path 'test.db', got '%s'", conf.DB)
	}

	if conf.Logger.Format != "json" {
		t.Fatalf("Expected logger format 'json', got '%s'", conf.Logger.Format)
	}

	if conf.Logger.Level != "info" {
		t.Fatalf("Expected logger level 'info', got '%s'", conf.Logger.Level)
	}

	if conf.Logger.Output != "discard" {
		t.Fatalf("Expected logger output 'discard', got '%s'", conf.Logger.Output)
	}
}

func TestParseNonExistentFile(t *testing.T) {
	_, err := Parse("non-existent-file.yaml")
	if err != nil {
		t.Errorf("Expected no error when parsing non-existent file, got %+v", err)
	}
}
