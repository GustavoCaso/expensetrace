package util

import "time"

func GetMonthDates(month int, year int) (time.Time, time.Time) {
	goMonth := time.Month(month)

	today := time.Now()
	currentLocation := today.Location()

	var y int
	if year > 0 {
		y = year
	} else {
		y = today.Year()
	}

	firstOfMonth := time.Date(y, goMonth, 1, 0, 0, 0, 0, currentLocation)
	lastOfMonth := firstOfMonth.AddDate(0, 1, 0).Add(time.Nanosecond * -1)

	return firstOfMonth, lastOfMonth
}

func GetYearDates(year int) (time.Time, time.Time) {
	today := time.Now()
	currentLocation := today.Location()

	firstOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, currentLocation)
	lastOfYear := time.Date(year, 12, 31, 0, 0, 0, 0, currentLocation)

	return firstOfYear, lastOfYear
}
