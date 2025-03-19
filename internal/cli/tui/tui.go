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

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Exit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
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

	altscreen bool

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

	reportsTable := newReports(reports, width, height)

	log.Printf("initial model created. width: %d height: %d\n", width, height)

	return model{
		reports:   reports,
		altscreen: false,

		keyMap: defaultKeyMap(),

		reportsTable: reportsTable,
		focusReport:  newfocusReport(0, 0),
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
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		log.Printf("Window change event. width: %d height: %d\n", msg.Width, msg.Height)
	case tea.KeyMsg:
		log.Printf("KeyMsg %s \n", msg.String())
		switch {
		case key.Matches(msg, m.keyMap.Enter):
			var cmd tea.Cmd
			m.altscreen, cmd = m.altscreenToggle()
			return m, cmd
		case key.Matches(msg, m.keyMap.Up), key.Matches(msg, m.keyMap.Down):
			if !m.altscreen {
				m.reportsTable, _ = m.reportsTable.Update(msg)
				m.focusReport = m.focusReport.UpdateTable(m.reports[m.reportsTable.Cursor()], m.width/2, m.height/2)
			} else {
				m.focusReport, _ = m.focusReport.Update(msg)
			}
		case key.Matches(msg, m.keyMap.Exit):
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	var main string
	log.Printf("View() width: %d height: %d\n", m.width, m.height)
	helpView := m.help.View(m.keyMap)
	log.Printf("help height: %d \n", lipgloss.Height(helpView))

	if !m.altscreen {
		main = m.reportsTable.View()
		log.Printf("reports table height: %d \n", lipgloss.Height(main))

		main = baseStyle.Width(m.width).Render(main)
		log.Printf("reports table with border height: %d \n", lipgloss.Height(main))
	} else {
		main = m.focusReport.View()
		log.Printf("focus report height: %d \n", lipgloss.Height(main))

		main = baseStyle.Width(m.width).Render(main)
		log.Printf("focus report with border height: %d \n", lipgloss.Height(main))
	}

	main = lipgloss.JoinVertical(lipgloss.Top, main, helpView)
	log.Printf("main height: %d \n", lipgloss.Height(main))

	return main
}

func (m model) altscreenToggle() (bool, tea.Cmd) {
	var cmd tea.Cmd
	if m.altscreen {
		cmd = tea.ExitAltScreen
	} else {
		cmd = tea.EnterAltScreen
	}

	return !m.altscreen, cmd
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
