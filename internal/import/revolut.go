package importutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var revolutTransformers = []transformer{
	func(v string, entry *entry) error {
		if v == "CHARGE" {
			entry.charge = true
		}
		return nil
	}, // Type
	nil, // Product
	func(v string, entry *entry) error { // Started Date
		t, err := time.Parse("2006-01-02 15:04:05", v)
		if err != nil {
			return err
		}
		entry.date = t
		return nil
	},
	nil, // Completed Date
	func(v string, entry *entry) error { // Description
		s := strings.ToLower(v)
		entry.description = s
		return nil
	},
	func(v string, entry *entry) error { // Amount
		matches := re.FindStringSubmatch(v)
		if len(matches) == 0 {
			return fmt.Errorf("amount regex did not find any matches")
		}

		amount := matches[amountIndex]
		decimal := matches[decimalIndex]

		parsedAmount, err := strconv.ParseInt(fmt.Sprintf("%s%s", amount, decimal), 10, 64)
		if err != nil {
			return err
		}

		if strings.HasPrefix(v, "-") || strings.HasPrefix(v, "−") {
			parsedAmount = parsedAmount * -1
		}

		if parsedAmount != 0 {
			entry.amount = parsedAmount
		}

		return nil
	},
	func(v string, entry *entry) error { // Fee
		matches := re.FindStringSubmatch(v)
		if len(matches) == 0 {
			return fmt.Errorf("amount regex did not find any matches")
		}

		amount := matches[amountIndex]
		decimal := matches[decimalIndex]

		parsedAmount, err := strconv.ParseInt(fmt.Sprintf("%s%s", amount, decimal), 10, 64)
		if err != nil {
			return err
		}

		if strings.HasPrefix(v, "-") || strings.HasPrefix(v, "−") {
			parsedAmount = parsedAmount * -1
		}

		if parsedAmount != 0 {
			if entry.amount != 0 {
				entry.amount -= parsedAmount
			} else {
				entry.amount = parsedAmount
			}
		}
		if entry.charge {
			entry.amount = entry.amount * -1
		}
		return nil
	},
	func(v string, entry *entry) error { // Currency
		entry.currency = v
		return nil
	},
	nil, // State
	nil, // Balance
}
