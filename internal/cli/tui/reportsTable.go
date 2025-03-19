package tui

import (
	"log"
	"sort"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type reportsTable struct {
	width  int
	height int
	table  table.Model
}

func newReports(reports []wrapper, width, height int) reportsTable {
	columns := createColumns(width)

	sort.SliceStable(reports, func(i, j int) bool {
		return reports[i].report.StartDate.After(reports[j].report.StartDate)
	})

	rows := []table.Row{}

	for _, report := range reports {
		rows = append(rows, report.ToRow())
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height/2),
	)

	return reportsTable{
		width:  width,
		height: height,
		table:  t,
	}
}

func (r reportsTable) Cursor() int {
	return r.table.Cursor()
}

func (r reportsTable) Update(msg tea.Msg) (reportsTable, tea.Cmd) {
	var cmd tea.Cmd
	log.Printf("update reports table. %s\n", msg.(tea.KeyMsg).String())
	r.table.Focus()
	r.table, cmd = r.table.Update(msg)
	return r, cmd
}

func (r reportsTable) View() string {
	return r.table.View()
}

func createColumns(width int) []table.Column {
	w := width / 4

	return []table.Column{
		{Title: "Month", Width: w},
		{Title: "Income", Width: w},
		{Title: "Spending", Width: w},
		{Title: "SavingsPercentage", Width: w},
	}
}
