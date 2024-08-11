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

func (ex Expense) AmountWithSign() int64 {
	if ex.Type == ChargeType {
		return -(ex.Amount)
	}

	return ex.Amount
}
