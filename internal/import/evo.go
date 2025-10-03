package importutil

import (
	"fmt"
	"regexp"
	"strconv"
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

		entry.amount = parsedAmount
		return nil
	},
	func(v string, entry *entry) error {
		entry.currency = v
		return nil
	},
	nil,
	nil,
}
