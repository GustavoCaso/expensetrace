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

func (w wrapper) ToFocusRows() []table.Row {
	rows := make([]table.Row, len(w.report.Categories))

	for i, category := range w.report.Categories {
		rows[i] = table.Row{
			category.Name,
			fmt.Sprintf("%s€", util.FormatMoney(category.Amount, ".", ",")),
		}
	}

	return rows
}
