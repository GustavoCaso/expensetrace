package tui

import (
	"database/sql"
	"flag"
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

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

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

type keymap struct {
	Enter key.Binding
	Up    key.Binding
	Down  key.Binding
	Exit  key.Binding
}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Exit}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter}, // first column
		{k.Exit},                // second column
	}
}

func defaultKeyMap() keymap {
	return keymap{
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

	keyMap keymap

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

		keyMap: defaultKeyMap(),

		reportsTable: reportsTable,
		focusReport:  newfocusReport(),
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
		case key.Matches(msg, m.keyMap.Enter):
			m.focusMode = m.focusModeToggle()
			return m, nil
		case key.Matches(msg, m.keyMap.Up), key.Matches(msg, m.keyMap.Down):
			if m.focusMode == focusedMain {
				m.reportsTable, cmd = m.reportsTable.Update(msg)
				m.focusReport = m.focusReport.UpdateTable(m.reports[m.reportsTable.Cursor()], m.width/2)
			} else {
				m.focusReport, cmd = m.focusReport.Update(msg)
			}
		case key.Matches(msg, m.keyMap.Exit):
			return m, tea.Quit
		}
	}

	m.reportsTable = m.reportsTable.UpdateDimensions(m.width, m.height/2)
	m.focusReport = m.focusReport.UpdateDimensions(m.width/2, m.height/2)

	return m, cmd
}

func (m model) View() string {
	var main string
	helpView := m.help.View(m.keyMap)

	if m.focusMode == focusedMain {
		main = baseStyle.Width(m.width).Render(m.reportsTable.View())
	} else {
		main = baseStyle.Width(m.width).Render(m.focusReport.View())
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

	p := tea.NewProgram(initialModel(db, w, h), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running TUI: %v", err)
	}
}
