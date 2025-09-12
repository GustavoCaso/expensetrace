package tui

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"

	"github.com/GustavoCaso/expensetrace/internal/cli"
	"github.com/GustavoCaso/expensetrace/internal/logger"
	"github.com/GustavoCaso/expensetrace/internal/matcher"
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/storage"
	"github.com/GustavoCaso/expensetrace/internal/util"
)

const (
	layoutSplitRatio = 2
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

type focusState int

const (
	focusedMain focusState = iota
	focusedDetail
)

const (
	numberOfPanels = 2
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

	reportsTable reportsTable
	focusReport  focusReport
	help         help.Model

	reportsKeyMap reportsKeymap
	focusKeymap   focusKeymap

	focusMode focusState

	width  int
	height int
}

func initialModel(storage storage.Storage, width int, height int) (model, error) {
	now := time.Now()
	month := now.Month()
	year := now.Year()

	reports, err := generateReports(storage, month, year)
	if err != nil {
		return model{}, err
	}

	return model{
		reports:   reports,
		focusMode: focusedMain,

		reportsKeyMap: reportsKeyMap(),
		focusKeymap:   focusKeyMap(),

		reportsTable: newReports(reports, width),
		focusReport:  newfocusReport(width/layoutSplitRatio, height/layoutSplitRatio),
		help:         help.New(),

		width:  width,
		height: height,
	}, nil
}

func generateReports(storage storage.Storage, month time.Month, year int) ([]wrapper, error) {
	reports := []wrapper{}
	skipYear := false
	timeMonth := month
	ex, err := storage.GetFirstExpense(context.Background())
	if err != nil {
		return reports, err
	}

	lastMonth := ex.Date().Month()
	lastYear := ex.Date().Year()

	for year >= lastYear {
		if timeMonth == time.January {
			skipYear = true
		}

		firstDay, lastDay := util.GetMonthDates(int(timeMonth), year)

		expenses, expensesErr := storage.GetExpensesFromDateRange(context.Background(), firstDay, lastDay)

		if expensesErr != nil {
			return reports, expensesErr
		}

		report, reportError := report.Generate(context.Background(), firstDay, lastDay, storage, expenses, "monthly")

		if reportError != nil {
			return reports, reportError
		}

		reports = append(reports, wrapper{
			report: report,
		})

		if skipYear {
			year--
			timeMonth = time.December
			skipYear = false
			continue
		}

		if year == lastYear && timeMonth == lastMonth {
			break
		}

		timeMonth--
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

	m.reportsTable.UpdateDimensions(m.width, m.height/numberOfPanels)
	m.focusReport.UpdateDimensions(m.width/numberOfPanels, m.height/numberOfPanels)

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

func (c tuiCommand) Run(storage storage.Storage, _ *matcher.Matcher, _ *logger.Logger) error {
	w, h, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		return fmt.Errorf("failed to get terminal size: %w", err)
	}

	if len(os.Getenv("EXPENSETRACE_DEBUG")) > 0 {
		f, logErr := tea.LogToFile("debug.log", "debug")
		if logErr != nil {
			return fmt.Errorf("failed to log to file: %w", logErr)
		}
		defer f.Close()
	}

	m, err := initialModel(storage, w, h)

	if err != nil {
		return fmt.Errorf("failed to create initia model: %w", err)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err = p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}
