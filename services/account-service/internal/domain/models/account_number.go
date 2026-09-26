package models

import (
	"fmt"
	"regexp"
)

// Номер счёта: балансовый префикс (5) + цифровой код валюты (3) + контрольный ключ (1) + серийник (11).
const (
	// BALANCE_PREFIX_LENGTH = 5
	CHECK_DIGIT_POSITION = 8
	SERIAL_LENGTH        = 11
	MAX_SERIAL           = 99_999_999_999
)

var balancePrefixFormat = regexp.MustCompile(`^[0-9]{5}$`)

// checkWeights - веса контрольного ключа по методике ЦБ, повторяются по кругу.
var checkWeights = [3]int{7, 1, 3}

// NumberGenerator собирает номера счетов с балансовым префиксом из конфига.
type NumberGenerator struct {
	balancePrefix string
}

func NewNumberGenerator(balancePrefix string) (*NumberGenerator, error) {
	if !balancePrefixFormat.MatchString(balancePrefix) {
		return nil, fmt.Errorf("balance prefix %q: expected 5 digits", balancePrefix)
	}

	return &NumberGenerator{balancePrefix: balancePrefix}, nil
}

// Build собирает номер из валюты и номера счета.
//
//	40817   810   4    00000000001
//	│      │    │         │
//	│      │    │         └─ серийник, 11 цифр (позиции 9–19)
//	│      │    └─ контрольный ключ, 1 цифра (позиция 8)
//	│      └─ цифровой код валюты ISO 4217, 3 цифры (позиции 5–7)
//	└─ балансовый префикс из конфига, 5 цифр (позиции 0–4)
func (g *NumberGenerator) Build(currency Currency, serial int64) (string, error) {
	numeric, ok := currency.NumericCode()
	if !ok {
		return "", fmt.Errorf("currency %s: unknown numeric code", currency)
	}

	if serial <= 0 || serial > MAX_SERIAL {
		return "", fmt.Errorf("serial %d is out of range 1..%d", serial, int64(MAX_SERIAL))
	}

	digits := []byte(fmt.Sprintf("%s%s0%0*d", g.balancePrefix, numeric, SERIAL_LENGTH, serial))
	digits[CHECK_DIGIT_POSITION] = '0' + byte(checkDigit(digits))

	return string(digits), nil
}

// checkDigit считает ключ по весам 7-1-3 без БИК: вес позиции ключа равен 3, отсюда множитель 3.
func checkDigit(digits []byte) int {
	withoutKey := make([]byte, len(digits))
	copy(withoutKey, digits)
	withoutKey[CHECK_DIGIT_POSITION] = '0'

	return (weightedSum(withoutKey) % 10) * 3 % 10
}

func weightedSum(digits []byte) int {
	sum := 0

	for i, d := range digits {
		sum += int(d-'0') * checkWeights[i%len(checkWeights)] % 10
	}

	return sum
}
