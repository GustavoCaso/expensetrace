package config

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParse(t *testing.T) {
	// Create a temporary test config file
	config := Config{
		DB: "test.db",
		Categories: Categories{
			Expense: []Category{
				{
					Name:    "Food",
					Pattern: "restaurant|food|grocery",
				},
				{
					Name:    "Transport",
					Pattern: "uber|taxi|transit",
				},
			},
			Income: []Category{
				{
					Name:    "Salary",
					Pattern: "salary|income",
				},
			},
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

	// Verify expense categories
	// Expenses categories
	if len(conf.Categories.Expense) != len(config.Categories.Expense) {
		t.Fatalf(
			"Expected %d expense categories, got %d",
			len(config.Categories.Expense),
			len(conf.Categories.Expense),
		)
	}

	for i, expected := range config.Categories.Expense {
		if conf.Categories.Expense[i] != expected {
			t.Errorf("Category[%d] = %+v, want %+v", i, conf.Categories.Expense[i], expected)
		}
	}

	// Expenses categories
	if len(conf.Categories.Income) != len(config.Categories.Income) {
		t.Fatalf("Expected %d income categories, got %d", len(config.Categories.Income), len(conf.Categories.Income))
	}

	for i, expected := range config.Categories.Income {
		if conf.Categories.Income[i] != expected {
			t.Errorf("Category[%d] = %+v, want %+v", i, conf.Categories.Income[i], expected)
		}
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

	// Verify expense categories
	if len(conf.Categories.Expense) != 0 {
		t.Fatalf("Expected 0 expense categories, got %d", len(conf.Categories.Expense))
	}

	// Verify income categories
	if len(conf.Categories.Income) != 0 {
		t.Fatalf("Expected 0 income categories, got %d", len(conf.Categories.Income))
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
