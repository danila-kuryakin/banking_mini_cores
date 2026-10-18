package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

const (
	TEST_BROKER   = "localhost:29092"
	TEST_TOPIC    = "ledger.events"
	TEST_GROUP_ID = "antifraud"
)

// validConfig возвращает новый валидный конфиг для каждого кейса
func validConfig() *Config {
	cfg := &Config{Currencies: []string{"RUB", "USD", "EUR"}}
	cfg.Kafka.Brokers = []string{TEST_BROKER}
	cfg.Kafka.Topics = []string{TEST_TOPIC}
	cfg.Kafka.GroupID = TEST_GROUP_ID
	return cfg
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name           string
		mutate         func(cfg *Config)
		wantCurrencies []models.Currency
		wantErr        string
	}{
		{
			name:           "valid config",
			mutate:         func(cfg *Config) {},
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "currencies normalized and empty skipped",
			mutate:         func(cfg *Config) { cfg.Currencies = []string{" rub", "usd", ""} },
			wantCurrencies: []models.Currency{"RUB", "USD"},
		},
		{
			name:           "single lowercase currency",
			mutate:         func(cfg *Config) { cfg.Currencies = []string{"eur"} },
			wantCurrencies: []models.Currency{"EUR"},
		},
		{
			name:           "source order preserved",
			mutate:         func(cfg *Config) { cfg.Currencies = []string{"USD", "EUR", "RUB"} },
			wantCurrencies: []models.Currency{"USD", "EUR", "RUB"},
		},
		{
			name:           "topic with surrounding spaces",
			mutate:         func(cfg *Config) { cfg.Kafka.Topics = []string{" ledger.events "} },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "topics with empty elements and one valid",
			mutate:         func(cfg *Config) { cfg.Kafka.Topics = []string{"", TEST_TOPIC, " "} },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "brokers with blank element and one valid",
			mutate:         func(cfg *Config) { cfg.Kafka.Brokers = []string{" ", TEST_BROKER} },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "required acks none",
			mutate:         func(cfg *Config) { cfg.Kafka.RequiredAcks = "none" },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "required acks leader",
			mutate:         func(cfg *Config) { cfg.Kafka.RequiredAcks = "leader" },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "required acks all",
			mutate:         func(cfg *Config) { cfg.Kafka.RequiredAcks = "all" },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "start offset first",
			mutate:         func(cfg *Config) { cfg.Kafka.StartOffset = "first" },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "start offset last",
			mutate:         func(cfg *Config) { cfg.Kafka.StartOffset = "last" },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "producer topic empty consumer topics set",
			mutate:         func(cfg *Config) { cfg.Kafka.Topic = "" },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "zero consumer server and postgres",
			mutate:         func(cfg *Config) { cfg.Consumer = Consumer{} },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "request timeout zero",
			mutate:         func(cfg *Config) { cfg.RequestTimeout = 0 },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "request timeout negative",
			mutate:         func(cfg *Config) { cfg.RequestTimeout = -time.Second },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "backoff zero",
			mutate:         func(cfg *Config) { cfg.Consumer = Consumer{MinBackoff: 0, MaxBackoff: 0} },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "backoff negative",
			mutate:         func(cfg *Config) { cfg.Consumer = Consumer{MinBackoff: -time.Second, MaxBackoff: -time.Second} },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:           "min backoff greater than max",
			mutate:         func(cfg *Config) { cfg.Consumer = Consumer{MinBackoff: 10 * time.Second, MaxBackoff: time.Second} },
			wantCurrencies: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:    "currencies nil",
			mutate:  func(cfg *Config) { cfg.Currencies = nil },
			wantErr: "currencies",
		},
		{
			name:    "currencies empty",
			mutate:  func(cfg *Config) { cfg.Currencies = []string{} },
			wantErr: "currencies",
		},
		{
			name:    "currencies only blank elements",
			mutate:  func(cfg *Config) { cfg.Currencies = []string{"", " "} },
			wantErr: "currencies",
		},
		{
			name:    "currencies duplicate after normalization",
			mutate:  func(cfg *Config) { cfg.Currencies = []string{"RUB", "rub"} },
			wantErr: "currencies",
		},
		{
			name:    "currency code too short",
			mutate:  func(cfg *Config) { cfg.Currencies = []string{"RU"} },
			wantErr: "currencies",
		},
		{
			name:    "currency code too long",
			mutate:  func(cfg *Config) { cfg.Currencies = []string{"RUBL"} },
			wantErr: "currencies",
		},
		{
			name:    "currency code digits",
			mutate:  func(cfg *Config) { cfg.Currencies = []string{"123"} },
			wantErr: "currencies",
		},
		{
			name:    "brokers nil",
			mutate:  func(cfg *Config) { cfg.Kafka.Brokers = nil },
			wantErr: "kafka",
		},
		{
			name:    "brokers empty",
			mutate:  func(cfg *Config) { cfg.Kafka.Brokers = []string{} },
			wantErr: "kafka",
		},
		{
			name:    "brokers single blank",
			mutate:  func(cfg *Config) { cfg.Kafka.Brokers = []string{" "} },
			wantErr: "kafka",
		},
		{
			name:    "brokers only blank elements",
			mutate:  func(cfg *Config) { cfg.Kafka.Brokers = []string{"", "  "} },
			wantErr: "kafka",
		},
		{
			name:    "required acks unknown",
			mutate:  func(cfg *Config) { cfg.Kafka.RequiredAcks = "some" },
			wantErr: "kafka",
		},
		{
			name:    "required acks wrong case",
			mutate:  func(cfg *Config) { cfg.Kafka.RequiredAcks = "ALL" },
			wantErr: "kafka",
		},
		{
			name:    "start offset unknown",
			mutate:  func(cfg *Config) { cfg.Kafka.StartOffset = "middle" },
			wantErr: "kafka",
		},
		{
			name:    "topics nil",
			mutate:  func(cfg *Config) { cfg.Kafka.Topics = nil },
			wantErr: "topics",
		},
		{
			name:    "topics empty",
			mutate:  func(cfg *Config) { cfg.Kafka.Topics = []string{} },
			wantErr: "topics",
		},
		{
			name:    "topics only blank elements",
			mutate:  func(cfg *Config) { cfg.Kafka.Topics = []string{" ", ""} },
			wantErr: "topics",
		},
		{
			name: "producer topic set consumer topics nil",
			mutate: func(cfg *Config) {
				cfg.Kafka.Topic = TEST_TOPIC
				cfg.Kafka.Topics = nil
			},
			wantErr: "topics",
		},
		{
			name:    "group id empty",
			mutate:  func(cfg *Config) { cfg.Kafka.GroupID = "" },
			wantErr: "group_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)

			err := cfg.Validate()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				// R6: при ошибке набор валют не заполняется
				assert.Equal(t, models.CurrencySet{}, cfg.CurrencySet)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCurrencies, cfg.CurrencySet.List())
		})
	}
}

// Несколько ошибок сразу: порядок проверок не специфицирован, проверяем только факт ошибки и R6
func TestConfig_Validate_MultipleErrors(t *testing.T) {
	cfg := validConfig()
	cfg.Currencies = nil
	cfg.Kafka.Brokers = nil
	cfg.Kafka.Topics = nil
	cfg.Kafka.GroupID = ""

	err := cfg.Validate()

	require.Error(t, err)
	assert.Equal(t, models.CurrencySet{}, cfg.CurrencySet)
}

func TestConfig_Validate_Repeated(t *testing.T) {
	cfg := validConfig()

	require.NoError(t, cfg.Validate())
	first := cfg.CurrencySet.List()
	require.NoError(t, cfg.Validate())

	assert.Equal(t, []models.Currency{"RUB", "USD", "EUR"}, first)
	assert.Equal(t, first, cfg.CurrencySet.List())
}

// Инвариант п.8: каждая валюта из List() содержится в наборе
func TestConfig_Validate_CurrencySetContainsListed(t *testing.T) {
	tests := []struct {
		name       string
		currencies []string
	}{
		{name: "three currencies", currencies: []string{"RUB", "USD", "EUR"}},
		{name: "normalized with blank", currencies: []string{" rub", "usd", ""}},
		{name: "single currency", currencies: []string{"eur"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Currencies = tt.currencies

			require.NoError(t, cfg.Validate())

			for _, currency := range cfg.CurrencySet.List() {
				assert.True(t, cfg.CurrencySet.Contains(currency), "currency %s not contained", currency)
			}
		})
	}
}
