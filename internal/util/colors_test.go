package util

import (
	"testing"

	"github.com/fatih/color"
)

func TestColorOutput(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		colorOptions   []string
		expectedLength int
	}{
		{
			name:           "single color",
			text:           "test text",
			colorOptions:   []string{"red"},
			expectedLength: len("test text"),
		},
		{
			name:           "multiple colors",
			text:           "test text",
			colorOptions:   []string{"red", "bold"},
			expectedLength: len("test text"),
		},
		{
			name:           "background color",
			text:           "test text",
			colorOptions:   []string{"bgRed"},
			expectedLength: len("test text"),
		},
		{
			name:           "invalid color option",
			text:           "test text",
			colorOptions:   []string{"invalid"},
			expectedLength: len("test text"),
		},
		{
			name:           "empty text",
			text:           "",
			colorOptions:   []string{"red"},
			expectedLength: 0,
		},
		{
			name:           "no color options",
			text:           "test text",
			colorOptions:   []string{},
			expectedLength: len("test text"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ColorOutput(tt.text, tt.colorOptions...)
			if len(result) != tt.expectedLength {
				t.Errorf("ColorOutput() length = %v, want %v", len(result), tt.expectedLength)
			}
		})
	}
}

func TestColorOptions(t *testing.T) {
	expectedOptions := map[string]color.Attribute{
		"red":       color.FgHiRed,
		"green":     color.FgGreen,
		"underline": color.Underline,
		"bold":      color.Bold,
		"bgRed":     color.BgRed,
		"bgGreen":   color.BgGreen,
	}

	for name, expected := range expectedOptions {
		if actual, ok := colorsOptions[name]; !ok {
			t.Errorf("Missing color option: %s", name)
		} else if actual != expected {
			t.Errorf("Color option %s = %v, want %v", name, actual, expected)
		}
	}
}
