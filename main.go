package main

import (
	"database/sql"
	_ "database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	_ "log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ExpenseType int

const (
	ChargeType ExpenseType = iota
	IncomeType
)

var re = regexp.MustCompile(`(?P<charge>-)?(?P<amount>\d+)\.?(?P<decimal>\d*)`)
var chargeIndex = re.SubexpIndex("charge")
var amountIndex = re.SubexpIndex("amount")
var decimalIndex = re.SubexpIndex("decimal")

type Expense struct {
	Date        time.Time
	Description string
	Amount      uint16
	Decimal     uint16
	Type        ExpenseType
	Currency    string
}

func main() {
	argsLength := len(os.Args)

	if argsLength != 2 {
		panic("must provide a CSV file with your expenseses")
	}
	file, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	r := csv.NewReader(file)
	expenses := []Expense{}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if strings.Contains(record[0], "Fecha") {
			// We skip the first line of the CSV
			continue
		}

		t, err := time.Parse("02/01/2006", record[1])
		if err != nil {
			log.Fatal(err)
		}

		matches := re.FindStringSubmatch(record[3])
		if len(matches) == 0 {
			log.Fatal("Amount regex did not find any matches")
		}

		var et ExpenseType
		if matches[chargeIndex] == "-" {
			et = ChargeType
		} else {
			et = IncomeType
		}

		amount := matches[amountIndex]
		decimal := matches[decimalIndex]

		parsedAmount, err := strconv.ParseUint(amount, 10, 16)
		if err != nil {
			log.Fatal(err)
		}
		parsedDecimal, err := strconv.ParseUint(decimal, 10, 16)
		if err != nil {
			log.Fatal(err)
		}

		expense := Expense{
			Date:        t,
			Description: strings.ToLower(record[2]),
			Amount:      uint16(parsedAmount),
			Decimal:     uint16(parsedDecimal),
			Type:        et,
			Currency:    record[4],
		}

		expenses = append(expenses, expense)
	}

	fmt.Println(expenses)

	db, err := sql.Open("sqlite3", "expenses.db")

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	// Create table
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS expenses (id INTEGER PRIMARY KEY, amount INTEGER NOT NULL, decimal INTEGER NOT NULL,  description TEXT NOT NULL, expense_type INTEGER NOT NULL, date INTEGER NOT NULL, currency TEXT NOT NULL) STRICT;")
	if err != nil {
		log.Printf("Error in creating table: %s\n", err.Error())
		os.Exit(1)
	} else {
		log.Println("Successfully created table expenses!")
	}
	statement.Exec()

	// Insert records
	insertstmt, err := db.Prepare("INSERT INTO expenses(amount, decimal, description, expense_type, date, currency) values(?, ?, ?, ?, ?, ?)")

	for _, expense := range expenses {
		_, err := insertstmt.Exec(expense.Amount, expense.Decimal, expense.Description, expense.Type, expense.Date.Unix(), expense.Currency)
		if err != nil {
			log.Fatal(err)
		}
	}
}
