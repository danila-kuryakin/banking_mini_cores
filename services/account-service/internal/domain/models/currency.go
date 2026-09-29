package models

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Currency - буквенный код валюты ISO 4217 в верхнем регистре.
type Currency string

var currencyFormat = regexp.MustCompile(`^[A-Z]{3}$`)

// currencyNumericCodes - цифровые коды ISO 4217, они входят в номер счёта.
var currencyNumericCodes = map[Currency]string{
	"RUB": "810",
	"USD": "840",
	"EUR": "978",
}

// ParseCurrency приводит код к верхнему регистру и проверяет формат.
func ParseCurrency(raw string) (Currency, error) {
	code := Currency(strings.ToUpper(strings.TrimSpace(raw)))
	if !currencyFormat.MatchString(string(code)) {
		return "", fmt.Errorf("currency %q: expected a three-letter ISO 4217 code", raw)
	}

	return code, nil
}

// NumericCode отдаёт цифровой код валюты для номера счёта.
func (c Currency) NumericCode() (string, bool) {
	code, ok := currencyNumericCodes[c]

	return code, ok
}

// CurrencySet - валюты, в которых разрешено открывать счета.
type CurrencySet struct {
	items []Currency
}

// NewCurrencySet собирает список из конфига: без повторов и только валюты с известным цифровым кодом.
func NewCurrencySet(raw []string) (CurrencySet, error) {
	items := make([]Currency, 0, len(raw))

	for _, value := range raw {
		if strings.TrimSpace(value) == "" {
			continue
		}

		code, err := ParseCurrency(value)
		if err != nil {
			return CurrencySet{}, err
		}

		if _, ok := code.NumericCode(); !ok {
			return CurrencySet{}, fmt.Errorf("currency %s: unknown ISO 4217 numeric code", code)
		}

		if slices.Contains(items, code) {
			return CurrencySet{}, fmt.Errorf("currency %s is listed twice", code)
		}

		items = append(items, code)
	}

	if len(items) == 0 {
		return CurrencySet{}, fmt.Errorf("currency list is empty")
	}

	return CurrencySet{items: items}, nil
}

func (s CurrencySet) Contains(c Currency) bool {
	return slices.Contains(s.items, c)
}

func (s CurrencySet) List() []Currency {
	return slices.Clone(s.items)
}
