package tui

import (
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type focusReport struct {
	table  table.Model
	report wrapper
}

func newfocusReport() focusReport {
	return focusReport{
		table: table.New(),
	}
}

func (d focusReport) UpdateTable(report wrapper, width int) focusReport {
	columns := createFocusColumns(width)

	rows := report.ToFocusRows()

	table := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
	)

	return focusReport{
		table:  table,
		report: report,
	}
}

func (d focusReport) Update(msg tea.Msg) (focusReport, tea.Cmd) {
	var cmd tea.Cmd

	d.table.Focus()
	d.table, cmd = d.table.Update(msg)
	return d, cmd
}

func (d focusReport) UpdateDimensions(width, height int) focusReport {
	t := d.table
	t.SetColumns(createFocusColumns(width))
	t.SetWidth(width)
	t.SetHeight(height)

	return focusReport{
		table:  t,
		report: d.report,
	}
}

func (d focusReport) View() string {
	return d.table.View()
}

func (d focusReport) Cursor() int {
	return d.table.Cursor()
}

func (d focusReport) Categories() []report.Category {
	return d.report.Categories()
}

func createFocusColumns(width int) []table.Column {
	w := width / 2

	return []table.Column{
		{Title: "Category", Width: w},
		{Title: "Spending", Width: w},
	}
}
