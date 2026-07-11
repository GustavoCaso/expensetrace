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

	if conf.Port != defaultPort {
		t.Fatalf("Expected port '%s', got '%s'", defaultPort, conf.Port)
	}

	if conf.Timeout.String() != defaultTimeout.String() {
		t.Fatalf("Expected timeout '%s', got '%s'", defaultTimeout.String(), conf.Timeout.String())
	}
}

func TestParseENV(t *testing.T) {
	t.Setenv("EXPENSETRACE_DB", "test.db")
	t.Setenv("EXPENSETRACE_LOG_LEVEL", "info")
	t.Setenv("EXPENSETRACE_LOG_FORMAT", "json")
	t.Setenv("EXPENSETRACE_LOG_OUTPUT", "discard")
	t.Setenv("EXPENSETRACE_PORT", "8765")
	t.Setenv("EXPENSETRACE_TIMEOUT", "10s")

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

	if conf.Port != "8765" {
		t.Fatalf("Expected port '%s', got '%s'", "8765", conf.Port)
	}

	if conf.Timeout.String() != "10s" {
		t.Fatalf("Expected timeout '%s', got '%s'", "10s", conf.Timeout.String())
	}
}
