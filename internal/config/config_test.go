package config

import (
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	// Create a temporary test config file
	content := `
db: "test.db"
categories:
  - name: "Food"
    pattern: "restaurant|food|grocery"
  - name: "Transport"
    pattern: "uber|taxi|transit"
`
	tmpfile, err := os.CreateTemp(t.TempDir(), "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err = tmpfile.WriteString(content); err != nil {
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

	// Verify categories
	if len(conf.Categories) != 2 {
		t.Fatalf("Expected 2 categories, got %d", len(conf.Categories))
	}

	// Verify first category
	if conf.Categories[0].Name != "Food" {
		t.Errorf("Expected category name 'Food', got '%s'", conf.Categories[0].Name)
	}
	if conf.Categories[0].Pattern != "restaurant|food|grocery" {
		t.Errorf("Expected category pattern 'restaurant|food|grocery', got '%s'", conf.Categories[0].Pattern)
	}

	// Verify second category
	if conf.Categories[1].Name != "Transport" {
		t.Errorf("Expected category name 'Transport', got '%s'", conf.Categories[1].Name)
	}
	if conf.Categories[1].Pattern != "uber|taxi|transit" {
		t.Errorf("Expected category pattern 'uber|taxi|transit', got '%s'", conf.Categories[1].Pattern)
	}
}

func TestParseENV(t *testing.T) {
	// Create a temporary test config file
	content := ``
	tmpfile, err := os.CreateTemp(t.TempDir(), "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err = tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	if err = tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temporary file: %v", err)
	}

	t.Setenv("EXPENSETRACE_DB", "test.db")

	// Test parsing the config file
	conf, err := Parse(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify database path
	if conf.DB != "test.db" {
		t.Errorf("Expected DB path 'test.db', got '%s'", conf.DB)
	}

	// Verify categories
	if len(conf.Categories) != 0 {
		t.Fatalf("Expected 0 categories, got %d", len(conf.Categories))
	}
}

func TestParseValidate(t *testing.T) {
	// Create a temporary test config file
	content := ``
	tmpfile, err := os.CreateTemp(t.TempDir(), "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err = tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	if err = tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temporary file: %v", err)
	}

	// Test parsing the config file
	_, err = Parse(tmpfile.Name())
	if err == nil {
		t.Error("Expected error when parsing config with no values and no ENV variable set, got nil")
	}

	if err.Error() != "DB is not set" {
		t.Errorf("Expected error message 'DB is not set', got '%s'", err.Error())
	}
}

func TestParseNonExistentFile(t *testing.T) {
	_, err := Parse("non-existent-file.yaml")
	if err == nil {
		t.Error("Expected error when parsing non-existent file, got nil")
	}
}

func TestParseInvalidYAML(t *testing.T) {
	// Create a temporary test config file with invalid YAML
	content := `
db: "test.db"
categories:
  - name: "Food"
    pattern: "restaurant|food|grocery"
  - name: "Transport"
    pattern: "uber|taxi|transit"
    invalid: field: here
`
	tmpfile, err := os.CreateTemp(t.TempDir(), "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err = tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}
	if err = tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temporary file: %v", err)
	}

	// Test parsing the invalid config file
	_, err = Parse(tmpfile.Name())
	if err == nil {
		t.Error("Expected error when parsing invalid YAML, got nil")
	}
}
