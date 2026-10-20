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
		{name: "upper case as is", raw: "EUR", want: "EUR"},
		{name: "trim spaces and upper", raw: " rub ", want: "RUB"},
		{name: "mixed case", raw: "Usd", want: "USD"},
		{name: "trailing newline trimmed", raw: "rub\n", want: "RUB"},
		{name: "tabs trimmed", raw: "\tusd\t", want: "USD"},
		{name: "empty", raw: "", wantErr: true},
		{name: "only spaces", raw: "   ", wantErr: true},
		{name: "two letters", raw: "RU", wantErr: true},
		{name: "four letters", raw: "RUBL", wantErr: true},
		{name: "digit inside", raw: "R1B", wantErr: true},
		{name: "space inside", raw: "R B", wantErr: true},
		{name: "cyrillic", raw: "РУБ", wantErr: true},
		{name: "digits only", raw: "123", wantErr: true},
		{name: "dash inside", raw: "R-B", wantErr: true},
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
	}{
		{name: "spaces lower", raw: " rub "},
		{name: "mixed case", raw: "Usd"},
		{name: "upper", raw: "EUR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, err := ParseCurrency(tt.raw)
			require.NoError(t, err)
			second, err := ParseCurrency(string(first))
			require.NoError(t, err)
			assert.Equal(t, first, second)
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
		{name: "plain list", raw: []string{"RUB", "USD", "EUR"}, want: []Currency{"RUB", "USD", "EUR"}},
		{name: "normalize and skip blanks", raw: []string{" rub", "USD", "", "  ", "eur"}, want: []Currency{"RUB", "USD", "EUR"}},
		{name: "order preserved", raw: []string{"USD", "EUR", "RUB"}, want: []Currency{"USD", "EUR", "RUB"}},
		{name: "single element", raw: []string{"rub"}, want: []Currency{"RUB"}},
		{name: "duplicate after normalization", raw: []string{"RUB", "rub"}, wantErr: true},
		{name: "exact duplicate", raw: []string{"RUB", "RUB"}, wantErr: true},
		{name: "invalid code short", raw: []string{"RUB", "XX"}, wantErr: true},
		{name: "invalid code digit", raw: []string{"RUB", "R1B"}, wantErr: true},
		{name: "empty slice", raw: []string{}, wantErr: true},
		{name: "nil slice", raw: nil, wantErr: true},
		{name: "only blanks", raw: []string{"", " "}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCurrencySet(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, CurrencySet{}, got)
				assert.Len(t, got.List(), 0)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.List())
		})
	}
}

func TestCurrencySet_ListReturnsCopy(t *testing.T) {
	set, err := NewCurrencySet([]string{"RUB", "USD"})
	require.NoError(t, err)

	list := set.List()
	require.Len(t, list, 2)
	list[0] = "XXX"

	assert.Equal(t, []Currency{"RUB", "USD"}, set.List())
	assert.True(t, set.Contains("RUB"))
	assert.False(t, set.Contains("XXX"))
}

func TestCurrencySet_Contains(t *testing.T) {
	tests := []struct {
		name     string
		currency Currency
		want     bool
	}{
		{name: "first member", currency: "RUB", want: true},
		{name: "second member", currency: "USD", want: true},
		{name: "not in set", currency: "EUR", want: false},
		{name: "lower case not normalized", currency: "rub", want: false},
		{name: "leading space not normalized", currency: " RUB", want: false},
		{name: "empty", currency: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := NewCurrencySet([]string{"RUB", "USD"})
			require.NoError(t, err)
			assert.Equal(t, tt.want, set.Contains(tt.currency))
		})
	}
}

func TestCurrencySet_ZeroValue(t *testing.T) {
	var set CurrencySet
	assert.False(t, set.Contains("RUB"))
	assert.Len(t, set.List(), 0)
}

func TestCurrencySet_ContainsEveryListed(t *testing.T) {
	tests := []struct {
		name string
		raw  []string
	}{
		{name: "single", raw: []string{"RUB"}},
		{name: "normalized with blanks", raw: []string{" rub", "USD", "", "eur"}},
		{name: "custom order", raw: []string{"USD", "EUR", "RUB"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := NewCurrencySet(tt.raw)
			require.NoError(t, err)
			for _, c := range set.List() {
				assert.True(t, set.Contains(c), "currency %s", c)
			}
		})
	}
}
