package tui

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"sort"
	"time"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/report"
	"github.com/GustavoCaso/expensetrace/internal/util"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
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

func defaultKeyMap() keymap {
	return keymap{
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "explore"),
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

	table       table.Model
	focusReport focusReport

	keyMap keymap

	focusMode focusState

	width  int
	height int

	cursor int
	view   string
}

func initialModel(db *sql.DB, width int, height int) model {
	log.Println("Initializing TUI model...")
	expenses, err := expenseDB.GetExpenses(db)
	if err != nil {
		log.Fatalf("Unable to get expenses: %s", err.Error())
	}
	log.Printf("Loaded %d expenses from database", len(expenses))

	now := time.Now()
	month := now.Month()
	year := now.Year()

	reports, err := generateReports(db, month, year)

	columns := []table.Column{
		{Title: "Month", Width: width / 4},
		{Title: "Income", Width: width / 4},
		{Title: "Spending", Width: width / 4},
		{Title: "SavingsPercentage", Width: width / 4},
	}

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
		table.WithHeight(height),
	)

	t.SetHeight(height / 2)

	return model{
		reports: reports,
		keyMap:  defaultKeyMap(),
		table:   t,

		focusReport: newfocusReport(0, 0),
		focusMode:   focusedMain,

		width:  width,
		height: height,

		cursor: 0,
		view:   "list",
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
	log.Println("Initializing TUI...")
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Enter):
			m.focusMode = m.nextFocus()
		case key.Matches(msg, m.keyMap.Up), key.Matches(msg, m.keyMap.Down):
			switch m.focusMode {
			case focusedMain:
				m.table, _ = m.table.Update(msg)
				m.focusReport = m.focusReport.UpdateTable(m.reports[m.table.Cursor()], m.width/2, m.height)
			case focusedDetail:
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
	left := m.table.View()

	switch m.focusMode {
	case focusedMain:
		main = left
	case focusedDetail:
		borderStyle := baseStyle.Width(m.width / 2)
		disabledBorderStyle := borderStyle.BorderForeground(lipgloss.Color("241"))
		left = disabledBorderStyle.Render(left)
		right := m.focusReport.View()
		right = borderStyle.Render(right)

		main = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	return main
}

func (m model) nextFocus() focusState {
	switch m.focusMode {
	case focusedMain:
		return focusedDetail
	case focusedDetail:
		return focusedMain
	default:
		panic("invalid focus state")
	}
}

func (c tuiCommand) Run(db *sql.DB, matcher *category.Matcher) {
	log.Println("Starting TUI application...")
	defer db.Close()

	w, h, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		log.Fatalf("failed to get terminal size: %v", err)
	}

	p := tea.NewProgram(initialModel(db, w, h))
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running TUI: %v", err)
	}
	log.Println("TUI application terminated successfully")
}
