package util

import "testing"

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		thousand string
		decimal  string
		expected string
	}{
		{
			name:     "positive value with default separators",
			value:    1234567,
			thousand: ".",
			decimal:  ",",
			expected: "12.345,67",
		},
		{
			name:     "negative value with default separators",
			value:    -1234567,
			thousand: ".",
			decimal:  ",",
			expected: "-12.345,67",
		},
		{
			name:     "zero value",
			value:    0,
			thousand: ".",
			decimal:  ",",
			expected: "0,00",
		},
		{
			name:     "value less than 100",
			value:    99,
			thousand: ".",
			decimal:  ",",
			expected: "0,99",
		},
		{
			name:     "value with custom separators",
			value:    1234567,
			thousand: ",",
			decimal:  ".",
			expected: "12,345.67",
		},
		{
			name:     "large value",
			value:    1234567890,
			thousand: ".",
			decimal:  ",",
			expected: "12.345.678,90",
		},
		{
			name:     "value with no thousands separator",
			value:    1234,
			thousand: "",
			decimal:  ",",
			expected: "12,34",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMoney(tt.value, tt.thousand, tt.decimal)
			if result != tt.expected {
				t.Errorf("FormatMoney() = %v, want %v", result, tt.expected)
			}
		})
	}
}
