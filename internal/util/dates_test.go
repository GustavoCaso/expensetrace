package util

import (
	"testing"
	"time"
)

func TestGetMonthDates(t *testing.T) {
	tests := []struct {
		name          string
		month         int
		year          int
		expectedStart time.Time
		expectedEnd   time.Time
	}{
		{
			name:          "January 2024",
			month:         1,
			year:          2024,
			expectedStart: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.Local),
			expectedEnd:   time.Date(2024, time.January, 31, 23, 59, 59, 999999999, time.Local),
		},
		{
			name:          "February 2024 (leap year)",
			month:         2,
			year:          2024,
			expectedStart: time.Date(2024, time.February, 1, 0, 0, 0, 0, time.Local),
			expectedEnd:   time.Date(2024, time.February, 29, 23, 59, 59, 999999999, time.Local),
		},
		{
			name:          "December 2023",
			month:         12,
			year:          2023,
			expectedStart: time.Date(2023, time.December, 1, 0, 0, 0, 0, time.Local),
			expectedEnd:   time.Date(2023, time.December, 31, 23, 59, 59, 999999999, time.Local),
		},
		{
			name:          "Current year when year is 0",
			month:         1,
			year:          0,
			expectedStart: time.Date(time.Now().Year(), time.January, 1, 0, 0, 0, 0, time.Local),
			expectedEnd:   time.Date(time.Now().Year(), time.January, 31, 23, 59, 59, 999999999, time.Local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := GetMonthDates(tt.month, tt.year)

			if !start.Equal(tt.expectedStart) {
				t.Errorf("GetMonthDates() start = %v, want %v", start, tt.expectedStart)
			}
			if !end.Equal(tt.expectedEnd) {
				t.Errorf("GetMonthDates() end = %v, want %v", end, tt.expectedEnd)
			}
		})
	}
}

func TestGetYearDates(t *testing.T) {
	tests := []struct {
		name          string
		year          int
		expectedStart time.Time
		expectedEnd   time.Time
	}{
		{
			name:          "2024",
			year:          2024,
			expectedStart: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.Local),
			expectedEnd:   time.Date(2024, time.December, 31, 0, 0, 0, 0, time.Local),
		},
		{
			name:          "2023",
			year:          2023,
			expectedStart: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.Local),
			expectedEnd:   time.Date(2023, time.December, 31, 0, 0, 0, 0, time.Local),
		},
		{
			name:          "2020 (leap year)",
			year:          2020,
			expectedStart: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.Local),
			expectedEnd:   time.Date(2020, time.December, 31, 0, 0, 0, 0, time.Local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := GetYearDates(tt.year)

			if !start.Equal(tt.expectedStart) {
				t.Errorf("GetYearDates() start = %v, want %v", start, tt.expectedStart)
			}
			if !end.Equal(tt.expectedEnd) {
				t.Errorf("GetYearDates() end = %v, want %v", end, tt.expectedEnd)
			}
		})
	}
}
