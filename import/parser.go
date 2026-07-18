package importutil

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
)

// ParsedData represents the raw data extracted from a file.
type ParsedData struct {
	Headers []string   // Column headers/field names
	Rows    [][]string // Data rows (all values as strings)
	Format  string     // File format (csv or json)
}

// ParseFile parses a CSV or JSON file and extracts headers and rows
// without making assumptions about structure or field mapping.
func ParseFile(filename string, reader io.Reader) (*ParsedData, error) {
	fileFormat := path.Ext(filename)

	switch fileFormat {
	case ".csv":
		return parseCSV(reader)
	case ".json":
		return parseJSON(reader)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", fileFormat)
	}
}

// parseCSV reads CSV data and extracts headers and rows.
func parseCSV(reader io.Reader) (*ParsedData, error) {
	r := csv.NewReader(reader)

	// Read all records
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, errors.New("CSV file is empty")
	}

	// First row is headers
	headers := records[0]

	// Remaining rows are data
	rows := records[1:]

	if len(rows) == 0 {
		return nil, errors.New("CSV file has no data rows")
	}

	return &ParsedData{
		Headers: headers,
		Rows:    rows,
		Format:  "csv",
	}, nil
}

// parseJSON reads JSON data and extracts headers and rows.
// Expects an array of objects with consistent fields.
func parseJSON(reader io.Reader) (*ParsedData, error) {
	var data []map[string]interface{}

	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	if len(data) == 0 {
		return nil, errors.New("JSON file contains no records")
	}

	// Extract headers from first object's keys
	headers := make([]string, 0, len(data[0]))
	for key := range data[0] {
		headers = append(headers, key)
	}

	// Convert all records to string rows
	rows := make([][]string, 0, len(data))
	for _, record := range data {
		row := make([]string, len(headers))
		for i, header := range headers {
			// Convert any type to string
			if val, ok := record[header]; ok && val != nil {
				row[i] = fmt.Sprintf("%v", val)
			} else {
				row[i] = ""
			}
		}
		rows = append(rows, row)
	}

	return &ParsedData{
		Headers: headers,
		Rows:    rows,
		Format:  "json",
	}, nil
}

// GetPreviewRows returns the first N rows for preview purposes.
func (p *ParsedData) GetPreviewRows(n int) [][]string {
	if n >= len(p.Rows) {
		return p.Rows
	}
	return p.Rows[:n]
}

// GetTotalRows returns the total number of data rows.
func (p *ParsedData) GetTotalRows() int {
	return len(p.Rows)
}
