package tui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focusView int

const (
	categoryView focusView = iota
	expensesView
)

const (
	focusColumns = 2
)

type focusReport struct {
	category table.Model
	expenses []table.Model
	report   wrapper

	width  int
	height int

	view focusView
}

func newfocusReport(width, height int) focusReport {
	return focusReport{
		view:     categoryView,
		category: table.Model{},
		expenses: []table.Model{},
		width:    width,
		height:   height,
	}
}

func (d *focusReport) SetReport(report wrapper) {
	d.report = report
	d.category = table.New(
		table.WithColumns(createCategoryColumns(d.width)),
		table.WithRows(report.ToFocusRows()),
	)

	categories := report.Categories()
	expensesTables := make([]table.Model, len(categories))

	for i, category := range categories {
		expensesTables[i] = table.New(
			table.WithColumns(createExpenseColumns(d.width)),
			table.WithRows(report.ExpensesToRow(category.Expenses, category.Amount < 0)),
		)
	}

	d.expenses = expensesTables
}

func (d *focusReport) focusModeToggle() {
	switch d.view {
	case categoryView:
		d.view = expensesView
	case expensesView:
		d.view = categoryView
	default:
		panic("invalid focus state")
	}
}

func (d focusReport) Update(msg tea.Msg) (focusReport, tea.Cmd) {
	if d.view == categoryView {
		d.category.Focus()
		category, cmd := d.category.Update(msg)

		d.category = category

		return d, cmd
	}
	d.category.Blur()
	expenseTable := d.expenses[d.category.Cursor()]
	expenseTable.Focus()

	expensesTableUpdated, cmd := expenseTable.Update(msg)
	d.expenses[d.category.Cursor()] = expensesTableUpdated

	return d, cmd
}

func (d *focusReport) UpdateDimensions(width, height int) {
	d.width = width
	d.height = height
}

func (d focusReport) View() string {
	categoryTable := d.category
	categoryTable.SetColumns(createCategoryColumns(d.width))
	categoryTable.SetWidth(d.width)
	categoryTable.SetHeight(d.height)

	expenseTable := d.expenses[d.category.Cursor()]
	expenseTable.SetColumns(createExpenseColumns(d.width))
	expenseTable.SetWidth(d.width)
	expenseTable.SetHeight(d.height)

	if d.view == categoryView {
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			focusedModelStyle.Render(categoryTable.View()),
			modelStyle.Render(expenseTable.View()),
		)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		modelStyle.Render(categoryTable.View()),
		focusedModelStyle.Render(expenseTable.View()),
	)
}

func createCategoryColumns(width int) []table.Column {
	w := width / focusColumns

	return []table.Column{
		{Title: "Category", Width: w},
		{Title: "Spending", Width: w},
	}
}

func createExpenseColumns(width int) []table.Column {
	w := width / focusColumns

	return []table.Column{
		{Title: "Description", Width: w},
		{Title: "Amount", Width: w},
	}
}
