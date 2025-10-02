package importutil

import (
	"strings"
	"testing"
)

func TestParseCSV(t *testing.T) {
	csvData := `source,date,description,amount,currency
Bank A,01/01/2024,Coffee shop,-5.50,USD
Bank B,02/01/2024,Salary,2500.00,USD
Bank A,03/01/2024,Grocery,-45.20,EUR`

	reader := strings.NewReader(csvData)
	parsed, err := ParseFile("test.csv", reader)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Check headers
	expectedHeaders := []string{"source", "date", "description", "amount", "currency"}
	if len(parsed.Headers) != len(expectedHeaders) {
		t.Fatalf("Expected %d headers, got %d", len(expectedHeaders), len(parsed.Headers))
	}
	for i, h := range expectedHeaders {
		if parsed.Headers[i] != h {
			t.Errorf("Header[%d] = %q, want %q", i, parsed.Headers[i], h)
		}
	}

	// Check rows
	if len(parsed.Rows) != 3 {
		t.Fatalf("Expected 3 rows, got %d", len(parsed.Rows))
	}

	// Check first row
	expectedRow := []string{"Bank A", "01/01/2024", "Coffee shop", "-5.50", "USD"}
	for i, val := range expectedRow {
		if parsed.Rows[0][i] != val {
			t.Errorf("Row[0][%d] = %q, want %q", i, parsed.Rows[0][i], val)
		}
	}

	// Check format
	if parsed.Format != "csv" {
		t.Errorf("Format = %q, want %q", parsed.Format, "csv")
	}

	// Check total rows
	if parsed.GetTotalRows() != 3 {
		t.Errorf("GetTotalRows() = %d, want 3", parsed.GetTotalRows())
	}
}

func TestParseJSON(t *testing.T) {
	jsonData := `[
		{
			"source": "Bank A",
			"date": "2024-01-01T00:00:00Z",
			"description": "Coffee shop",
			"amount": -5.50,
			"currency": "USD"
		},
		{
			"source": "Bank B",
			"date": "2024-01-02T00:00:00Z",
			"description": "Salary",
			"amount": 2500.00,
			"currency": "USD"
		}
	]`

	reader := strings.NewReader(jsonData)
	parsed, err := ParseFile("test.json", reader)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Check that we have headers (keys from JSON)
	if len(parsed.Headers) != 5 {
		t.Fatalf("Expected 5 headers, got %d", len(parsed.Headers))
	}

	// Check rows
	if len(parsed.Rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(parsed.Rows))
	}

	// Check format
	if parsed.Format != "json" {
		t.Errorf("Format = %q, want %q", parsed.Format, "json")
	}
}

func TestParseCSVEmpty(t *testing.T) {
	csvData := ``

	reader := strings.NewReader(csvData)
	_, err := ParseFile("test.csv", reader)
	if err == nil {
		t.Fatal("Expected error for empty CSV")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("Expected 'empty' error, got: %v", err)
	}
}

func TestParseCSVNoData(t *testing.T) {
	csvData := `source,date,description,amount,currency`

	reader := strings.NewReader(csvData)
	_, err := ParseFile("test.csv", reader)
	if err == nil {
		t.Fatal("Expected error for CSV with no data rows")
	}
	if !strings.Contains(err.Error(), "no data rows") {
		t.Errorf("Expected 'no data rows' error, got: %v", err)
	}
}

func TestParseJSONEmpty(t *testing.T) {
	jsonData := `[]`

	reader := strings.NewReader(jsonData)
	_, err := ParseFile("test.json", reader)
	if err == nil {
		t.Fatal("Expected error for empty JSON array")
	}
	if !strings.Contains(err.Error(), "no records") {
		t.Errorf("Expected 'no records' error, got: %v", err)
	}
}

func TestParseJSONInvalid(t *testing.T) {
	jsonData := `{invalid json`

	reader := strings.NewReader(jsonData)
	_, err := ParseFile("test.json", reader)
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestParseUnsupportedFormat(t *testing.T) {
	reader := strings.NewReader("some data")
	_, err := ParseFile("test.txt", reader)
	if err == nil {
		t.Fatal("Expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported file format") {
		t.Errorf("Expected 'unsupported file format' error, got: %v", err)
	}
}

func TestGetPreviewRows(t *testing.T) {
	csvData := `source,date,description,amount,currency
Bank A,01/01/2024,Coffee,-5.50,USD
Bank A,02/01/2024,Lunch,-12.00,USD
Bank A,03/01/2024,Dinner,-25.00,USD
Bank A,04/01/2024,Breakfast,-8.00,USD
Bank A,05/01/2024,Snack,-3.00,USD
Bank A,06/01/2024,Dessert,-6.00,USD`

	reader := strings.NewReader(csvData)
	parsed, err := ParseFile("test.csv", reader)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Test preview with limit less than total rows
	preview := parsed.GetPreviewRows(3)
	if len(preview) != 3 {
		t.Errorf("GetPreviewRows(3) returned %d rows, want 3", len(preview))
	}

	// Test preview with limit greater than total rows
	preview = parsed.GetPreviewRows(10)
	if len(preview) != 6 {
		t.Errorf("GetPreviewRows(10) returned %d rows, want 6 (all rows)", len(preview))
	}

	// Test preview with limit equal to total rows
	preview = parsed.GetPreviewRows(6)
	if len(preview) != 6 {
		t.Errorf("GetPreviewRows(6) returned %d rows, want 6", len(preview))
	}
}

func TestParseCSVWithCommasInValues(t *testing.T) {
	csvData := `source,date,description,amount,currency
"Bank A","01/01/2024","Coffee, Tea, and Snacks","-5.50","USD"
Bank B,02/01/2024,Salary,2500.00,USD`

	reader := strings.NewReader(csvData)
	parsed, err := ParseFile("test.csv", reader)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Check that comma within quotes is preserved
	if parsed.Rows[0][2] != "Coffee, Tea, and Snacks" {
		t.Errorf("Description = %q, want 'Coffee, Tea, and Snacks'", parsed.Rows[0][2])
	}
}
