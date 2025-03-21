package tui

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/util"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

var modelStyle = lipgloss.NewStyle().
	Align(lipgloss.Center, lipgloss.Center).
	BorderStyle(lipgloss.HiddenBorder()).
	Border(lipgloss.NormalBorder())

var focusedModelStyle = lipgloss.NewStyle().
	Align(lipgloss.Center, lipgloss.Center).
	BorderStyle(lipgloss.NormalBorder()).
	Border(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("69"))

type tuiCommand struct{}

func NewCommand() cli.Command {
	return tuiCommand{}
}

func (c tuiCommand) Description() string {
	return "Interactive terminal user interface"
}

func (c tuiCommand) SetFlags(*flag.FlagSet) {}

type focusState int

const (
	focusedMain focusState = iota
	focusedDetail
)

type reportsKeymap struct {
	Enter key.Binding
	Up    key.Binding
	Down  key.Binding
	Exit  key.Binding
}

func (k reportsKeymap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Exit}
}

func (k reportsKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter}, // first column
		{k.Exit},                // second column
	}
}

type focusKeymap struct {
	Enter key.Binding
	Up    key.Binding
	Down  key.Binding
	Tab   key.Binding
	Exit  key.Binding
}

func (k focusKeymap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Tab, k.Exit}
}

func (k focusKeymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Tab}, // first column
		{k.Exit},                       // second column
	}
}

func focusKeyMap() focusKeymap {
	return focusKeymap{
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "toggle detail view"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle expenses view"),
		),
		Exit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "exit"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
	}
}

func reportsKeyMap() reportsKeymap {
	return reportsKeymap{
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "toggle detail view"),
		),
		Exit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "exit"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
	}
}

type model struct {
	reports []wrapper

	current *wrapper

	reportsTable reportsTable
	focusReport  focusReport
	help         help.Model

	reportsKeyMap reportsKeymap
	focusKeymap   focusKeymap

	focusMode focusState

	width  int
	height int
}

func initialModel(db *sql.DB, width int, height int) model {
	now := time.Now()
	month := now.Month()
	year := now.Year()

	reports, err := generateReports(db, month, year)
	if err != nil {
		log.Fatalf("Unable to generate reports: %s", err.Error())
	}

	reportsTable := newReports(reports, width)

	return model{
		reports:   reports,
		focusMode: focusedMain,

		reportsKeyMap: reportsKeyMap(),
		focusKeymap:   focusKeyMap(),

		reportsTable: reportsTable,
		focusReport:  newfocusReport(width/2, height/2),
		help:         help.New(),

		width:  width,
		height: height,
	}
}

func generateReports(db *sql.DB, month time.Month, year int) ([]wrapper, error) {
	reports := []wrapper{}
	skipYear := false
	timeMonth := time.Month(month)
	ex, err := expenseDB.GetFirstExpense(db)
	if err != nil {
		return reports, err
	}

	lastMonth := ex.Date.Month()
	lastYear := ex.Date.Year()

	for year >= lastYear {
		if timeMonth == time.January {
			skipYear = true
		}

		firstDay, lastDay := util.GetMonthDates(int(timeMonth), year)

		expenses, err := expenseDB.GetExpensesFromDateRange(db, firstDay, lastDay)

		if err != nil {
			return reports, err
		}

		result := report.Generate(firstDay, lastDay, expenses, "monthly")

		reports = append(reports, wrapper{
			report: result,
		})

		if skipYear {
			year = year - 1
			timeMonth = time.December
			skipYear = false
			continue
		}

		if year == lastYear && timeMonth == lastMonth {
			break
		}

		timeMonth = timeMonth - 1
	}

	return reports, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetWidth(msg.Width)
		m.SetHeight(msg.Height)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.reportsKeyMap.Enter):
			m.focusMode = m.focusModeToggle()
			return m, nil
		case key.Matches(msg, m.reportsKeyMap.Up), key.Matches(msg, m.reportsKeyMap.Down):
			if m.focusMode == focusedMain {
				m.reportsTable, cmd = m.reportsTable.Update(msg)
				m.focusReport.SetReport(m.reports[m.reportsTable.Cursor()])
			} else {
				m.focusReport, cmd = m.focusReport.Update(msg)
			}
		case key.Matches(msg, m.focusKeymap.Tab):
			m.focusReport.focusModeToggle()
			return m, nil
		case key.Matches(msg, m.reportsKeyMap.Exit):
			return m, tea.Quit
		}
	}

	m.reportsTable.UpdateDimensions(m.width, m.height/2)
	m.focusReport.UpdateDimensions(m.width/2, m.height/2)

	return m, cmd
}

func (m model) View() string {
	var main string
	var helpView string

	if m.focusMode == focusedMain {
		helpView = m.help.View(m.reportsKeyMap)
		main = m.reportsTable.View()
	} else {
		helpView = m.help.View(m.focusKeymap)
		main = m.focusReport.View()
	}

	main = lipgloss.JoinVertical(lipgloss.Top, main, helpView)

	return main
}

func (m model) focusModeToggle() focusState {
	switch m.focusMode {
	case focusedMain:
		return focusedDetail
	case focusedDetail:
		return focusedMain
	default:
		panic("invalid focus state")
	}
}

func (m *model) SetHeight(height int) {
	m.height = height
}

func (m *model) SetWidth(width int) {
	m.width = width
}

func (c tuiCommand) Run(db *sql.DB, matcher *category.Matcher) {
	defer db.Close()

	w, h, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		log.Fatalf("failed to get terminal size: %v", err)
	}

	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	p := tea.NewProgram(initialModel(db, w, h), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running TUI: %v", err)
	}
}
