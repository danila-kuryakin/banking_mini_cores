package models

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

type Currency string

var currencyFormat = regexp.MustCompile(`^[A-Z]{3}$`)

func ParseCurrency(raw string) (Currency, error) {
	code := Currency(strings.ToUpper(strings.TrimSpace(raw)))
	if !currencyFormat.MatchString(string(code)) {
		return "", fmt.Errorf("currency %q: expected a three-letter ISO 4217 code", raw)
	}

	return code, nil
}

type CurrencySet struct {
	items []Currency
}

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
