package tui

import (
	"log"

	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type focusReport struct {
	width  int
	height int
	table  table.Model
	report wrapper
}

func newfocusReport(width, height int) focusReport {
	return focusReport{
		width:  width,
		height: height,
		table:  table.New(),
	}
}

func (d focusReport) UpdateTable(report wrapper, width, height int) focusReport {
	columns := []table.Column{
		{Title: "Category", Width: width / 2},
		{Title: "Spending", Width: width / 2},
	}

	rows := report.ToFocusRows()

	log.Printf("updatetable focus report table. width: %d height: %d \n", width, height)

	table := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(height),
		table.WithWidth(width),
	)

	return focusReport{
		table:  table,
		width:  width,
		height: height,
		report: report,
	}
}

func (d focusReport) Update(msg tea.Msg) (focusReport, tea.Cmd) {
	var cmd tea.Cmd
	log.Printf("update focus report table. %s\n", msg.(tea.KeyMsg).String())
	d.table.Focus()
	d.table, cmd = d.table.Update(msg)
	return d, cmd
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
