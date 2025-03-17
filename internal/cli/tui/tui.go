package tui

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tuiCommand struct{}

func NewCommand() cli.Command {
	return tuiCommand{}
}

func (c tuiCommand) Description() string {
	return "Interactive terminal user interface"
}

func (c tuiCommand) SetFlags(*flag.FlagSet) {}

type model struct {
	expenses []*expenseDB.Expense
	cursor   int
	view     string
}

func initialModel(db *sql.DB) model {
	log.Println("Initializing TUI model...")
	expenses, err := expenseDB.GetExpenses(db)
	if err != nil {
		log.Fatalf("Unable to get expenses: %s", err.Error())
	}
	log.Printf("Loaded %d expenses from database", len(expenses))

	return model{
		expenses: expenses,
		cursor:   0,
		view:     "list",
	}
}

func (m model) Init() tea.Cmd {
	log.Println("Initializing TUI...")
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		log.Printf("Key pressed: %s", key)

		switch key {
		case "ctrl+c", "q":
			log.Println("Quit command received")
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				log.Printf("Moving cursor up from %d to %d", m.cursor, m.cursor-1)
				m.cursor--
			} else {
				log.Println("Cursor already at top")
			}
		case "down", "j":
			if m.cursor < len(m.expenses)-1 {
				log.Printf("Moving cursor down from %d to %d", m.cursor, m.cursor+1)
				m.cursor++
			} else {
				log.Println("Cursor already at bottom")
			}
		default:
			log.Printf("Unhandled key: %s", key)
		}
	case tea.WindowSizeMsg:
		log.Printf("Window size changed: %dx%d", msg.Width, msg.Height)
	}
	return m, nil
}

func (m model) View() string {
	s := strings.Builder{}
	s.WriteString("ExpenseTrace TUI\n\n")

	// Style definitions
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		PaddingLeft(4)

	itemStyle := lipgloss.NewStyle().PaddingLeft(4)

	// Header
	s.WriteString(titleStyle.Render("Your Expenses\n\n"))

	// List of expenses
	for i, expense := range m.expenses {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		amount := fmt.Sprintf("%.2f€", float64(expense.Amount)/100)
		if expense.Type == expenseDB.ChargeType {
			amount = "-" + amount
		}

		item := fmt.Sprintf("%s %s | %s | %s\n",
			cursor,
			expense.Date.Format("2006-01-02"),
			expense.Description,
			amount,
		)

		if m.cursor == i {
			s.WriteString(itemStyle.Render(item))
		} else {
			s.WriteString(itemStyle.Render(item))
		}
	}

	s.WriteString("\nPress q to quit\n")

	return s.String()
}

func (c tuiCommand) Run(db *sql.DB, matcher *category.Matcher) {
	log.Println("Starting TUI application...")
	defer db.Close()

	p := tea.NewProgram(initialModel(db))
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running TUI: %v", err)
	}
	log.Println("TUI application terminated successfully")
}
