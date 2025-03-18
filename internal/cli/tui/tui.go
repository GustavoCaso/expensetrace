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

type model struct {
	reports map[int]map[int]wrapper

	current *wrapper

	table table.Model

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

	rows := []table.Row{}

	for _, reportsPerYear := range reports {
		for _, report := range reportsPerYear {
			rows = append(rows, report.ToRow())
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	t.SetHeight(height / 2)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return model{
		reports: reports,
		current: nil,

		table: t,

		width:  width,
		height: height,

		cursor: 0,
		view:   "list",
	}
}

func generateReports(db *sql.DB, month time.Month, year int) (map[int]map[int]wrapper, error) {
	reports := map[int]map[int]wrapper{}
	skipYear := false
	timeMonth := time.Month(month)
	for year > 2022 {
		if timeMonth == time.January {
			skipYear = true
		}

		firstDay, lastDay := util.GetMonthDates(int(timeMonth), year)

		expenses, err := expenseDB.GetExpensesFromDateRange(db, firstDay, lastDay)

		if err != nil {
			return reports, err
		}

		result := report.Generate(firstDay, lastDay, expenses, "monthly")

		report := wrapper{
			report: result,
		}

		_, ok := reports[year]

		if !ok {
			r := map[int]wrapper{}

			r[int(timeMonth)] = report

			reports[year] = r
		} else {
			reports[year][int(timeMonth)] = report
		}

		if skipYear {
			year = year - 1
			timeMonth = time.December
			skipYear = false
			continue
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
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
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
