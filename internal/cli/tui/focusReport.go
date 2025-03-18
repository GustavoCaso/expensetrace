package tui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type focusReport struct {
	width  int
	height int
	table  table.Model
}

func newfocusReport(width, height int) focusReport {
	return focusReport{
		width:  width,
		height: height,
		table:  table.New(table.WithHeight(height)),
	}
}

func (d focusReport) UpdateTable(report wrapper, width, height int) focusReport {
	columns := []table.Column{
		{Title: "Category", Width: width / 2},
		{Title: "Spending", Width: width / 2},
	}

	rows := report.ToFocusRows()

	table := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(height),
	)

	return focusReport{
		table:  table,
		width:  width,
		height: height,
	}
}

func (d focusReport) Update(msg tea.Msg) (focusReport, tea.Cmd) {
	var cmd tea.Cmd
	d.table.Focus()
	d.table, cmd = d.table.Update(msg)
	return d, cmd
}

func (d focusReport) View() string {
	return d.table.View()
}
