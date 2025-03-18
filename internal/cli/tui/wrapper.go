package tui

import (
	"fmt"

	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/util"
	"github.com/charmbracelet/bubbles/table"
)

type wrapper struct {
	report report.Report
}

func (w wrapper) ToRow() table.Row {
	return table.Row{
		w.report.Title,
		fmt.Sprintf("%s€", util.FormatMoney(w.report.Income, ".", ",")),
		fmt.Sprintf("%s€", util.FormatMoney(w.report.Spending, ".", ",")),
		fmt.Sprintf("%.2f%%", w.report.SavingsPercentage),
	}
}
