package config

import (
	"testing"
)

func TestParseDefaults(t *testing.T) {
	// Test parsing the config file
	conf := Parse()
	// Verify database path
	if conf.DBFile != defaultDBFile {
		t.Errorf("Expected DB path '%s', got '%s'", defaultDBFile, conf.DBFile)
	}

	if conf.Logger.Format != defaultLogFormat {
		t.Fatalf("Expected logger format '%s', got '%s'", defaultDBFile, conf.Logger.Format)
	}

	if conf.Logger.Level != defaultLogLevel {
		t.Fatalf("Expected logger level '%s', got '%s'", defaultLogLevel, conf.Logger.Level)
	}

	if conf.Logger.Output != defaultLogOutput {
		t.Fatalf("Expected logger output '%s', got '%s'", defaultLogOutput, conf.Logger.Output)
	}
}

func TestParseENV(t *testing.T) {
	t.Setenv("EXPENSETRACE_DB", "test.db")
	t.Setenv("EXPENSETRACE_LOG_LEVEL", "info")
	t.Setenv("EXPENSETRACE_LOG_FORMAT", "json")
	t.Setenv("EXPENSETRACE_LOG_OUTPUT", "discard")

	// Test parsing the config file
	conf := Parse()
	// Verify database path
	if conf.DBFile != "test.db" {
		t.Errorf("Expected DB path 'test.db', got '%s'", conf.DBFile)
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
