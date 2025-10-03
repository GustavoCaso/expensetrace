package importutil

import (
	"regexp"
	"strings"
	"time"
)

var cardPaymentString = regexp.MustCompile("pago en el dia tj-")

var evoTransformers = []transformer{
	func(v string, entry *entry) error {
		t, err := time.Parse("02/01/2006", v)
		if err != nil {
			return err
		}
		entry.date = t
		return nil
	},
	nil,
	func(v string, entry *entry) error {
		s := strings.ToLower(v)
		result := cardPaymentString.ReplaceAllString(s, "")
		entry.description = result
		return nil
	},
	func(v string, entry *entry) error {
		amount, err := parseAmount(v)

		if err != nil {
			return err
		}

		entry.amount = amount
		return nil
	},
	func(v string, entry *entry) error {
		entry.currency = v
		return nil
	},
	nil,
	nil,
}
