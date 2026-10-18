package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
)

const (
	TEST_CUSTOMER_UPSTREAM = "customer-service:50052"
	TEST_LEDGER_UPSTREAM   = "ledger-service:50056"
	TEST_BALANCE_PREFIX    = "40817"
)

// validBaseConfig собирает валидную базу; Server, Postgres и RequestTimeout остаются нулевыми.
func validBaseConfig() *Config {
	return &Config{
		Currencies:      []string{"RUB", "USD", "EUR"},
		DefaultCurrency: "RUB",
		AccountNumber:   AccountNumber{BalancePrefix: TEST_BALANCE_PREFIX},
		Upstreams: Upstreams{
			Customer: TEST_CUSTOMER_UPSTREAM,
			Ledger:   TEST_LEDGER_UPSTREAM,
		},
	}
}

func TestConfig_Validate_Success(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(c *Config)
		wantList    []models.Currency
		wantDefault models.Currency
		buildCur    models.Currency
		buildSerial int64
		wantNumber  string
	}{
		{
			name:        "valid base",
			mutate:      func(c *Config) {},
			wantList:    []models.Currency{"RUB", "USD", "EUR"},
			wantDefault: "RUB",
			buildCur:    "RUB",
			buildSerial: 1,
			wantNumber:  "40817810400000000001",
		},
		{
			name: "lowercase and spaces normalized",
			mutate: func(c *Config) {
				c.Currencies = []string{"rub", " usd "}
				c.DefaultCurrency = "usd"
			},
			wantList:    []models.Currency{"RUB", "USD"},
			wantDefault: "USD",
			buildCur:    "RUB",
			buildSerial: 1,
			wantNumber:  "40817810400000000001",
		},
		{
			name:        "default with surrounding spaces",
			mutate:      func(c *Config) { c.DefaultCurrency = " RUB " },
			wantList:    []models.Currency{"RUB", "USD", "EUR"},
			wantDefault: "RUB",
			buildCur:    "RUB",
			buildSerial: 1,
			wantNumber:  "40817810400000000001",
		},
		{
			name:        "default lowercase",
			mutate:      func(c *Config) { c.DefaultCurrency = "rub" },
			wantList:    []models.Currency{"RUB", "USD", "EUR"},
			wantDefault: "RUB",
			buildCur:    "RUB",
			buildSerial: 1,
			wantNumber:  "40817810400000000001",
		},
		{
			name:        "source order preserved",
			mutate:      func(c *Config) { c.Currencies = []string{"EUR", "RUB"} },
			wantList:    []models.Currency{"EUR", "RUB"},
			wantDefault: "RUB",
			buildCur:    "RUB",
			buildSerial: 1,
			wantNumber:  "40817810400000000001",
		},
		{
			name:        "single currency",
			mutate:      func(c *Config) { c.Currencies = []string{"RUB"} },
			wantList:    []models.Currency{"RUB"},
			wantDefault: "RUB",
			buildCur:    "RUB",
			buildSerial: 1,
			wantNumber:  "40817810400000000001",
		},
		{
			name:        "empty entries skipped",
			mutate:      func(c *Config) { c.Currencies = []string{"RUB", "", "USD"} },
			wantList:    []models.Currency{"RUB", "USD"},
			wantDefault: "RUB",
			buildCur:    "RUB",
			buildSerial: 1,
			wantNumber:  "40817810400000000001",
		},
		{
			name:        "prefix 40702",
			mutate:      func(c *Config) { c.AccountNumber.BalancePrefix = "40702" },
			wantList:    []models.Currency{"RUB", "USD", "EUR"},
			wantDefault: "RUB",
			buildCur:    "RUB",
			buildSerial: 12345,
			wantNumber:  "40702810900000012345",
		},
		{
			// ключ: 8*3 + 1*7 + 1*1 = 32 -> 2*3 = 6
			name:        "prefix all zeros",
			mutate:      func(c *Config) { c.AccountNumber.BalancePrefix = "00000" },
			wantList:    []models.Currency{"RUB", "USD", "EUR"},
			wantDefault: "RUB",
			buildCur:    "RUB",
			buildSerial: 1,
			wantNumber:  "00000810600000000001",
		},
		{
			name:        "zero request timeout",
			mutate:      func(c *Config) { c.RequestTimeout = 0 },
			wantList:    []models.Currency{"RUB", "USD", "EUR"},
			wantDefault: "RUB",
			buildCur:    "RUB",
			buildSerial: 1,
			wantNumber:  "40817810400000000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validBaseConfig()
			tt.mutate(cfg)

			require.NoError(t, cfg.Validate())

			assert.Equal(t, tt.wantList, cfg.CurrencySet.List())
			assert.Equal(t, tt.wantDefault, cfg.DefaultCurrencyCode)
			require.NotNil(t, cfg.NumberGenerator)
			accountNumber, err := cfg.NumberGenerator.Build(tt.buildCur, tt.buildSerial)
			require.NoError(t, err)
			assert.Equal(t, tt.wantNumber, accountNumber)
		})
	}
}

func TestConfig_Validate_Error(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(c *Config)
	}{
		{name: "empty customer upstream", mutate: func(c *Config) { c.Upstreams.Customer = "" }},
		{name: "empty ledger upstream", mutate: func(c *Config) { c.Upstreams.Ledger = "" }},
		{name: "both upstreams empty", mutate: func(c *Config) { c.Upstreams = Upstreams{} }},
		{name: "nil currencies", mutate: func(c *Config) { c.Currencies = nil }},
		{name: "empty currencies", mutate: func(c *Config) { c.Currencies = []string{} }},
		{name: "single empty entry", mutate: func(c *Config) { c.Currencies = []string{""} }},
		{name: "only blank entries", mutate: func(c *Config) { c.Currencies = []string{"", "  "} }},
		{name: "duplicate currency", mutate: func(c *Config) { c.Currencies = []string{"RUB", "USD", "RUB"} }},
		{name: "duplicate after normalization", mutate: func(c *Config) { c.Currencies = []string{"rub", "RUB"} }},
		{name: "currency two letters", mutate: func(c *Config) { c.Currencies = []string{"RUB", "RU"} }},
		{name: "currency four letters", mutate: func(c *Config) { c.Currencies = []string{"RUB", "RUBL"} }},
		{name: "currency with digit", mutate: func(c *Config) { c.Currencies = []string{"RUB", "R1B"} }},
		{name: "currency cyrillic", mutate: func(c *Config) { c.Currencies = []string{"RUB", "РУБ"} }},
		{name: "currency without numeric code", mutate: func(c *Config) { c.Currencies = []string{"RUB", "GEL"} }},
		{name: "empty default", mutate: func(c *Config) { c.DefaultCurrency = "" }},
		{name: "default two letters", mutate: func(c *Config) { c.DefaultCurrency = "RU" }},
		{name: "default with digit", mutate: func(c *Config) { c.DefaultCurrency = "R1B" }},
		{name: "default only spaces", mutate: func(c *Config) { c.DefaultCurrency = "   " }},
		{
			name: "default not in list",
			mutate: func(c *Config) {
				c.Currencies = []string{"RUB", "USD"}
				c.DefaultCurrency = "EUR"
			},
		},
		{name: "default GEL not in list", mutate: func(c *Config) { c.DefaultCurrency = "GEL" }},
		{
			name: "empty currencies and empty default",
			mutate: func(c *Config) {
				c.Currencies = nil
				c.DefaultCurrency = ""
			},
		},
		{name: "empty prefix", mutate: func(c *Config) { c.AccountNumber.BalancePrefix = "" }},
		{name: "prefix four digits", mutate: func(c *Config) { c.AccountNumber.BalancePrefix = "4081" }},
		{name: "prefix with letter", mutate: func(c *Config) { c.AccountNumber.BalancePrefix = "4081a" }},
		{name: "prefix six digits", mutate: func(c *Config) { c.AccountNumber.BalancePrefix = "408170" }},
		{name: "prefix fullwidth digits", mutate: func(c *Config) { c.AccountNumber.BalancePrefix = "４０８１７" }},
		{name: "prefix with minus", mutate: func(c *Config) { c.AccountNumber.BalancePrefix = "-4081" }},
		{name: "prefix trailing space", mutate: func(c *Config) { c.AccountNumber.BalancePrefix = "40817 " }},
		{name: "prefix leading space", mutate: func(c *Config) { c.AccountNumber.BalancePrefix = " 40817" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validBaseConfig()
			tt.mutate(cfg)

			require.Error(t, cfg.Validate())
		})
	}
}

func TestConfig_Validate_DefaultInCurrencySet(t *testing.T) {
	tests := []struct {
		name       string
		currencies []string
		def        string
	}{
		{name: "valid base", currencies: []string{"RUB", "USD", "EUR"}, def: "RUB"},
		{name: "lowercase with spaces", currencies: []string{"rub", " usd "}, def: "usd"},
		{name: "default with spaces", currencies: []string{"EUR", "RUB"}, def: " eur "},
		{name: "single currency", currencies: []string{"USD"}, def: "usd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validBaseConfig()
			cfg.Currencies = tt.currencies
			cfg.DefaultCurrency = tt.def

			require.NoError(t, cfg.Validate())

			assert.True(t, cfg.CurrencySet.Contains(cfg.DefaultCurrencyCode))
		})
	}
}
