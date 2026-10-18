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
		{name: "upper case", raw: "RUB", want: "RUB"},
		{name: "lower case", raw: "rub", want: "RUB"},
		{name: "space and tab around", raw: " usd\t", want: "USD"},
		{name: "mixed case", raw: "Eur", want: "EUR"},
		{name: "newlines around", raw: "\n gel \r\n", want: "GEL"},
		{name: "empty", raw: "", wantErr: true},
		{name: "only spaces", raw: "   ", wantErr: true},
		{name: "two letters", raw: "RU", wantErr: true},
		{name: "four letters", raw: "RUBL", wantErr: true},
		{name: "digit inside", raw: "R1B", wantErr: true},
		{name: "space inside", raw: "RU B", wantErr: true},
		{name: "cyrillic", raw: "РУБ", wantErr: true},
		{name: "underscore", raw: "R_B", wantErr: true},
		{name: "numeric iso code", raw: "643", wantErr: true},
		{name: "latin outside A-Z", raw: "RÜB", wantErr: true},
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

func TestNewCurrencySet(t *testing.T) {
	tests := []struct {
		name    string
		raw     []string
		want    []Currency
		wantErr bool
	}{
		{name: "normalizes and skips empty", raw: []string{"RUB", " usd ", "", "eur"}, want: []Currency{"RUB", "USD", "EUR"}},
		{name: "keeps input order", raw: []string{"EUR", "RUB"}, want: []Currency{"EUR", "RUB"}},
		{name: "single element", raw: []string{"rub"}, want: []Currency{"RUB"}},
		{name: "blank elements skipped", raw: []string{"", "RUB", "  \t"}, want: []Currency{"RUB"}},
		{name: "invalid code", raw: []string{"RUB", "XX"}, wantErr: true},
		{name: "duplicate different case", raw: []string{"RUB", "rub"}, wantErr: true},
		{name: "duplicate after trim", raw: []string{"RUB", " RUB "}, wantErr: true},
		{name: "nil slice", raw: nil, wantErr: true},
		{name: "empty slice", raw: []string{}, wantErr: true},
		{name: "only blank elements", raw: []string{"", "  "}, wantErr: true},
		{name: "space inside code", raw: []string{"RU B"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := NewCurrencySet(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				// при ошибке набор нулевой
				assert.Len(t, set.List(), 0)
				assert.False(t, set.Contains("RUB"))
				assert.False(t, set.Contains("XX"))
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
		{name: "present code", raw: []string{"RUB", "USD"}, currency: "USD", want: true},
		{name: "lower case not normalized", raw: []string{"RUB"}, currency: "rub", want: false},
		{name: "leading space not normalized", raw: []string{"RUB"}, currency: " RUB", want: false},
		{name: "absent code", raw: []string{"RUB"}, currency: "EUR", want: false},
		{name: "empty code", raw: []string{"RUB"}, currency: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := NewCurrencySet(tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.want, set.Contains(tt.currency))
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

func TestCurrencySet_ZeroValue(t *testing.T) {
	var set CurrencySet

	assert.False(t, set.Contains("RUB"))
	assert.False(t, set.Contains(""))
	assert.Len(t, set.List(), 0)
}

func TestParseCurrency_Idempotent(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "lower case", raw: "rub"},
		{name: "spaces around", raw: " EUR "},
		{name: "mixed case", raw: "Usd"},
		{name: "trailing tab", raw: "gbp\t"},
		{name: "already normalized", raw: "JPY"},
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

func TestCurrencySet_ContainsEveryListed(t *testing.T) {
	tests := []struct {
		name string
		raw  []string
	}{
		{name: "single", raw: []string{"RUB"}},
		{name: "lower case", raw: []string{"rub", "usd", "eur"}},
		{name: "with blanks", raw: []string{" eur ", "", "JPY"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := NewCurrencySet(tt.raw)
			require.NoError(t, err)

			for _, currency := range set.List() {
				assert.True(t, set.Contains(currency), "currency %s must be contained", currency)
			}
		})
	}
}
