package importutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var bankinterTransformers = []transformer{
	func(v string, entry *entry) error { // Fecha Contable
		t, err := time.Parse("02/01/2006", v)
		if err != nil {
			return err
		}
		entry.date = t
		return nil
	},
	nil, // Fecha Valor
	func(v string, entry *entry) error { // Descripcion
		s := strings.ToLower(v)
		entry.description = s
		return nil
	},
	func(v string, entry *entry) error { // Importe
		v = strings.ReplaceAll(v, "\"", "")
		v = strings.ReplaceAll(v, ",", "")

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
	func(_ string, entry *entry) error { // Saldo but we add the currency
		entry.currency = "EUR"
		return nil
	},
	nil,
	nil,
}
