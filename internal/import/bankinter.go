package importutil

import (
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

		amount, err := parseAmount(v)

		if err != nil {
			return err
		}

		entry.amount = amount
		return nil
	},
	func(_ string, entry *entry) error { // Saldo but we add the currency
		entry.currency = "EUR"
		return nil
	},
	nil,
	nil,
}
