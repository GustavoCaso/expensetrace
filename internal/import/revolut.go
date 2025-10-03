package importutil

import (
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
		amount, err := parseAmount(v)

		if err != nil {
			return err
		}

		entry.amount = amount

		return nil
	},
	func(v string, entry *entry) error { // Fee
		feeAmount, err := parseAmount(v)

		if err != nil {
			return err
		}

		if feeAmount != 0 {
			if entry.amount != 0 {
				entry.amount -= feeAmount
			} else {
				entry.amount = feeAmount
			}
		}
		if entry.charge {
			entry.amount *= -1
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
