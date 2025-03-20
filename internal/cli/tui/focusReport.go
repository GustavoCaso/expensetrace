package tui

import (
	"fmt"

	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/util"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/list"
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
	focusReport := d.table.View()
	expenses := d.Categories()[d.Cursor()].Expenses
	items := []string{}
	for _, expense := range expenses {
		items = append(items, fmt.Sprintf("%s | %s€", expense.Description, util.FormatMoney(expense.AmountWithSign(), ".", ",")))
	}
	l := list.New(items)
	listView := l.String()

	return lipgloss.JoinHorizontal(lipgloss.Top, focusReport, listView)
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
