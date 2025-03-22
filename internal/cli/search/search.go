package search

import (
	"bytes"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"sort"

	categoryPkg "github.com/GustavoCaso/expensetrace/internal/category"
	"github.com/GustavoCaso/expensetrace/internal/cli"
	expenseDB "github.com/GustavoCaso/expensetrace/internal/db"
	"github.com/GustavoCaso/expensetrace/internal/util"
	"github.com/fatih/color"
)

// content holds our static content.
//
//go:embed templates/*
var content embed.FS

type report struct {
	Categories map[string]category
	Verbose    bool
}

type category struct {
	name         string
	amount       int64
	categoryType expenseDB.ExpenseType
	expenses     []*expenseDB.Expense
}

func (c category) Display(verbose bool) string {
	value := c.amount

	var buffer = bytes.Buffer{}

	if value < 0 {
		buffer.WriteString(fmt.Sprintf("%s: %s€", c.name, colorOutput(util.FormatMoney(value, ".", ","), "red", "underline")))
	} else {
		buffer.WriteString(fmt.Sprintf("%s: %s€", c.name, colorOutput(util.FormatMoney(value, ".", ","), "green", "bold")))
	}

	if verbose {
		buffer.WriteString("\n")
		for _, ex := range c.expenses {
			buffer.WriteString(fmt.Sprintf("%s %s %s€\n", ex.Date.Format("2006-01-02"), ex.Description, util.FormatMoney(ex.Amount, ".", ",")))
		}
	}

	return buffer.String()
}

type searchCommand struct {
}

func NewCommand() cli.Command {
	return searchCommand{}
}

func (c searchCommand) Description() string {
	return "Search expenses"
}

var keyword string
var verbose bool

func (c searchCommand) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&keyword, "k", "", "keyword to use for the search")
	fs.BoolVar(&verbose, "v", false, "show verbose report output")
}

func (c searchCommand) Run(db *sql.DB, matcher *categoryPkg.Matcher) error {
	if keyword == "" {
		return fmt.Errorf("you must provide a keyword to use for the search")
	}

	expenses, err := expenseDB.SearchExpenses(db, keyword)
	if err != nil {
		return fmt.Errorf("enable to search the expenses DB: %w", err)
	}

	sort.Slice(expenses, func(i, j int) bool {
		return expenses[i].Date.Unix() < expenses[j].Date.Unix()
	})

	categories := make(map[string]category)

	for _, ex := range expenses {
		categoryName := expeseCategory(ex)

		c, ok := categories[categoryName]
		if ok {
			c.amount += ex.Amount
			c.expenses = append(c.expenses, ex)
			categories[categoryName] = c
		} else {
			c := category{
				amount:       ex.Amount,
				name:         categoryName,
				categoryType: ex.Type,
				expenses: []*expenseDB.Expense{
					ex,
				},
			}
			categories[categoryName] = c
		}
	}

	err = renderTemplate(os.Stdout, "report.tmpl", report{
		Categories: categories,
		Verbose:    verbose,
	})
	if err != nil {
		return fmt.Errorf("enable to render report: %w", err)
	}

	return nil
}

func expeseCategory(ex *expenseDB.Expense) string {
	if ex.CategoryID == 0 {
		if ex.Type == expenseDB.IncomeType {
			return "uncategorized income"
		} else {
			return "uncategorized charge"
		}
	}
	return "Needs to fix this"
}

var colorsOptions = map[string]color.Attribute{
	"red":       color.FgHiRed,
	"green":     color.FgGreen,
	"underline": color.Underline,
	"bold":      color.Bold,
	"bgRed":     color.BgRed,
	"bgGreen":   color.BgGreen,
}

func colorOutput(text string, colorOptions ...string) string {
	attributes := []color.Attribute{}
	for _, option := range colorOptions {
		if o, ok := colorsOptions[option]; ok {
			attributes = append(attributes, o)
		}

	}
	c := color.New(attributes...)
	return c.Sprint(text)
}

var templateFuncs = template.FuncMap{
	"formatMoney": util.FormatMoney,
	"colorOutput": colorOutput,
}

func renderTemplate(out io.Writer, templateName string, value interface{}) error {
	tmpl, err := content.ReadFile(path.Join("templates", templateName))
	if err != nil {
		return err
	}
	t := template.Must(template.New(templateName).Funcs(templateFuncs).Parse(string(tmpl)))
	err = t.Execute(out, value)
	if err != nil {
		return err
	}

	return nil
}
