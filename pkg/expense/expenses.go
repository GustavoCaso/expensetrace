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
	ID          int
	Date        time.Time
	Description string
	Amount      int64
	Type        ExpenseType
	Currency    string
	Category    string
}
