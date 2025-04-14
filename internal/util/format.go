package util

import "fmt"

const (
	decimalValue  = 100
	thousandValue = 1000
)

func FormatMoney(value int64, thousand, decimal string) string {
	var result string
	var isNegative bool

	if value < 0 {
		value *= -1
		isNegative = true
	}

	// apply the decimal separator
	result = fmt.Sprintf("%s%02d%s", decimal, value%decimalValue, result)
	value /= decimalValue

	// for each 3 dígits put a dot "."
	for value >= thousandValue {
		result = fmt.Sprintf("%s%03d%s", thousand, value%thousandValue, result)
		value /= thousandValue
	}

	if isNegative {
		return fmt.Sprintf("-%d%s", value, result)
	}

	return fmt.Sprintf("%d%s", value, result)
}
