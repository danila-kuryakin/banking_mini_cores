package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCurrency(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    Currency
		wantErr bool
	}{
		{name: "upper case", raw: "RUB", want: Currency("RUB")},
		{name: "lower case", raw: "rub", want: Currency("RUB")},
		{name: "spaces around", raw: " usd ", want: Currency("USD")},
		{name: "mixed case", raw: "Eur", want: Currency("EUR")},
		{name: "tab and newline around", raw: "\tRUB\n", want: Currency("RUB")},
		{name: "valid format without numeric code", raw: "GEL", want: Currency("GEL")},
		{name: "empty", raw: "", wantErr: true},
		{name: "only spaces", raw: "   ", wantErr: true},
		{name: "two letters", raw: "RU", wantErr: true},
		{name: "four letters", raw: "RUBL", wantErr: true},
		{name: "digit inside", raw: "R1B", wantErr: true},
		{name: "space inside", raw: "R B", wantErr: true},
		{name: "non latin letter", raw: "ÄBC", wantErr: true},
		{name: "full width letters", raw: "ＲＵＢ", wantErr: true},
		{name: "punctuation inside", raw: "R-B", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCurrency(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, Currency(""), got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseCurrency_Idempotent(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want Currency
	}{
		{name: "lower case", raw: "rub", want: Currency("RUB")},
		{name: "spaces and mixed case", raw: " Usd ", want: Currency("USD")},
		{name: "already normalized", raw: "EUR", want: Currency("EUR")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, err := ParseCurrency(tt.raw)
			require.NoError(t, err)
			second, err := ParseCurrency(string(first))
			require.NoError(t, err)
			assert.Equal(t, tt.want, first)
			assert.Equal(t, first, second)
		})
	}
}

func TestCurrency_NumericCode(t *testing.T) {
	tests := []struct {
		name     string
		currency Currency
		wantCode string
		wantOK   bool
	}{
		{name: "RUB", currency: Currency("RUB"), wantCode: "810", wantOK: true},
		{name: "USD", currency: Currency("USD"), wantCode: "840", wantOK: true},
		{name: "EUR", currency: Currency("EUR"), wantCode: "978", wantOK: true},
		{name: "unknown GEL", currency: Currency("GEL"), wantCode: "", wantOK: false},
		{name: "empty", currency: Currency(""), wantCode: "", wantOK: false},
		{name: "lower case not normalized", currency: Currency("rub"), wantCode: "", wantOK: false},
		{name: "leading space not normalized", currency: Currency(" RUB"), wantCode: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, ok := tt.currency.NumericCode()
			assert.Equal(t, tt.wantCode, code)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestNewCurrencySet(t *testing.T) {
	tests := []struct {
		name    string
		raw     []string
		want    []Currency
		wantErr bool
	}{
		{name: "normalizes and keeps order", raw: []string{"RUB", "usd", " EUR "}, want: []Currency{"RUB", "USD", "EUR"}},
		{name: "skips empty entries", raw: []string{"USD", "", "  ", "RUB"}, want: []Currency{"USD", "RUB"}},
		{name: "single entry", raw: []string{"EUR"}, want: []Currency{"EUR"}},
		{name: "nil list", raw: nil, wantErr: true},
		{name: "empty list", raw: []string{}, wantErr: true},
		{name: "only empty entries", raw: []string{"", " "}, wantErr: true},
		{name: "duplicate after normalization", raw: []string{"RUB", "rub"}, wantErr: true},
		{name: "exact duplicate", raw: []string{"USD", "USD"}, wantErr: true},
		{name: "currency without numeric code", raw: []string{"RUB", "GEL"}, wantErr: true},
		{name: "malformed entry", raw: []string{"RUB", "RU"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := NewCurrencySet(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				assert.False(t, set.Contains(Currency("RUB")))
				assert.Len(t, set.List(), 0)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, set.List())
		})
	}
}

func TestCurrencySet_Contains(t *testing.T) {
	tests := []struct {
		name     string
		raw      []string
		currency Currency
		want     bool
	}{
		{name: "present", raw: []string{"RUB", "USD"}, currency: Currency("RUB"), want: true},
		{name: "absent", raw: []string{"RUB", "USD"}, currency: Currency("EUR"), want: false},
		{name: "lower case not normalized", raw: []string{"RUB"}, currency: Currency("rub"), want: false},
		{name: "empty currency", raw: []string{"RUB"}, currency: Currency(""), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := NewCurrencySet(tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.want, set.Contains(tt.currency))
		})
	}
}

func TestCurrencySet_ZeroValue(t *testing.T) {
	var set CurrencySet

	assert.False(t, set.Contains(Currency("RUB")))
	assert.Len(t, set.List(), 0)
}

func TestCurrencySet_ListReturnsCopy(t *testing.T) {
	set, err := NewCurrencySet([]string{"RUB", "USD"})
	require.NoError(t, err)

	list := set.List()
	require.Len(t, list, 2)
	list[0] = Currency("XXX")

	assert.Equal(t, []Currency{"RUB", "USD"}, set.List())
	assert.True(t, set.Contains(Currency("RUB")))
	assert.False(t, set.Contains(Currency("XXX")))
}
