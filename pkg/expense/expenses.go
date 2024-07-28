package expense

import (
	"time"
)

type ExpenseType int

const (
	ChargeType ExpenseType = iota
	IncomeType
)

type Expense struct {
	Date        time.Time
	Description string
	Amount      uint16
	Decimal     uint16
	Type        ExpenseType
	Currency    string
	Category    string
}
