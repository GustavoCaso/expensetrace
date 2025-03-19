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
	var savingsPercentage string
	if w.report.Savings > 0 {
		savingsPercentage = util.ColorOutput(fmt.Sprintf("%.2f%%", w.report.SavingsPercentage), "green", "bold")
	} else {
		savingsPercentage = util.ColorOutput(fmt.Sprintf("%.2f%%", w.report.SavingsPercentage), "red", "underline")
	}

	return table.Row{
		w.report.Title,
		util.ColorOutput(fmt.Sprintf("%s€", util.FormatMoney(w.report.Income, ".", ",")), "green"),
		util.ColorOutput(fmt.Sprintf("%s€", util.FormatMoney(w.report.Spending, ".", ",")), "red", "underline"),
		savingsPercentage,
	}
}

func (w wrapper) ToFocusRows() []table.Row {
	rows := make([]table.Row, len(w.report.Categories))

	for i, category := range w.report.Categories {
		var categoryAmount string

		if category.Amount < 0 {
			categoryAmount = util.ColorOutput(fmt.Sprintf("%s€", util.FormatMoney(category.Amount, ".", ",")), "red", "underline")
		} else {
			categoryAmount = util.ColorOutput(fmt.Sprintf("%s€", util.FormatMoney(category.Amount, ".", ",")), "green")
		}

		rows[i] = table.Row{
			category.Name,
			categoryAmount,
		}
	}

	return rows
}

func (w wrapper) Categories() []report.Category {
	return w.report.Categories
}
