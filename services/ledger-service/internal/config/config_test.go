package config

import (
	"strings"
	"testing"

	platformconfig "github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_ACCOUNT_UPSTREAM   = "localhost:50055"
	TEST_ANTIFRAUD_UPSTREAM = "localhost:50057"
	TEST_KAFKA_BROKER       = "localhost:29092"
	TEST_KAFKA_TOPIC        = "ledger.events"
)

// validConfig возвращает новый валидный конфиг для каждого кейса
func validConfig() *Config {
	return &Config{
		Currencies: []string{"RUB", "USD", "EUR"},
		Upstreams:  Upstreams{Account: TEST_ACCOUNT_UPSTREAM, Antifraud: TEST_ANTIFRAUD_UPSTREAM},
		Kafka:      platformconfig.KafkaConfig{Brokers: []string{TEST_KAFKA_BROKER}, Topic: TEST_KAFKA_TOPIC},
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name         string
		mutate       func(cfg *Config)
		want         []models.Currency
		wantErr      bool
		wantContains []string
		wantPrefix   string
	}{
		{
			name:   "valid base config",
			mutate: func(cfg *Config) {},
			want:   []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:   "currencies normalized",
			mutate: func(cfg *Config) { cfg.Currencies = []string{" rub", "USD"} },
			want:   []models.Currency{"RUB", "USD"},
		},
		{
			name: "valid acks and offset",
			mutate: func(cfg *Config) {
				cfg.Kafka.RequiredAcks = "all"
				cfg.Kafka.StartOffset = "first"
			},
			want: []models.Currency{"RUB", "USD", "EUR"},
		},
		{
			name:         "empty account upstream",
			mutate:       func(cfg *Config) { cfg.Upstreams.Account = "" },
			wantErr:      true,
			wantContains: []string{"account-service"},
		},
		{
			name:         "empty antifraud upstream",
			mutate:       func(cfg *Config) { cfg.Upstreams.Antifraud = "" },
			wantErr:      true,
			wantContains: []string{"antifraud-service"},
		},
		{
			name:       "nil currencies",
			mutate:     func(cfg *Config) { cfg.Currencies = nil },
			wantErr:    true,
			wantPrefix: "currencies: ",
		},
		{
			name:       "duplicate currencies",
			mutate:     func(cfg *Config) { cfg.Currencies = []string{"RUB", "rub"} },
			wantErr:    true,
			wantPrefix: "currencies: ",
		},
		{
			name:       "invalid currency code",
			mutate:     func(cfg *Config) { cfg.Currencies = []string{"RUB", "XX"} },
			wantErr:    true,
			wantPrefix: "currencies: ",
		},
		{
			name:         "no kafka brokers",
			mutate:       func(cfg *Config) { cfg.Kafka.Brokers = nil },
			wantErr:      true,
			wantContains: []string{"kafka", "no brokers"},
		},
		{
			name:         "invalid required acks",
			mutate:       func(cfg *Config) { cfg.Kafka.RequiredAcks = "bad" },
			wantErr:      true,
			wantContains: []string{"kafka"},
		},
		{
			name:         "invalid start offset",
			mutate:       func(cfg *Config) { cfg.Kafka.StartOffset = "middle" },
			wantErr:      true,
			wantContains: []string{"kafka"},
		},
		{
			name:         "empty kafka topic",
			mutate:       func(cfg *Config) { cfg.Kafka.Topic = "" },
			wantErr:      true,
			wantContains: []string{"topic"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)

			err := cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				for _, part := range tt.wantContains {
					assert.Contains(t, err.Error(), part)
				}
				assert.True(t, strings.HasPrefix(err.Error(), tt.wantPrefix), "error %q must start with %q", err.Error(), tt.wantPrefix)
				// при ошибке набор валют не заполняется
				assert.Len(t, cfg.CurrencySet.List(), 0)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, cfg.CurrencySet.List())
		})
	}
}

func TestConfig_ValidateOrder(t *testing.T) {
	tests := []struct {
		name           string
		mutate         func(cfg *Config)
		wantContains   string
		wantNotContain string
		wantPrefix     string
	}{
		{
			name: "account checked first",
			mutate: func(cfg *Config) {
				cfg.Upstreams.Account = ""
				cfg.Upstreams.Antifraud = ""
				cfg.Currencies = nil
				cfg.Kafka.Brokers = nil
				cfg.Kafka.Topic = ""
			},
			wantContains:   "account-service",
			wantNotContain: "antifraud-service",
		},
		{
			name: "antifraud before currencies",
			mutate: func(cfg *Config) {
				cfg.Upstreams.Antifraud = ""
				cfg.Currencies = nil
				cfg.Kafka.Brokers = nil
				cfg.Kafka.Topic = ""
			},
			wantContains:   "antifraud-service",
			wantNotContain: "currencies: ",
		},
		{
			name: "currencies before kafka",
			mutate: func(cfg *Config) {
				cfg.Currencies = []string{"XX"}
				cfg.Kafka.Brokers = nil
				cfg.Kafka.Topic = ""
			},
			wantContains:   "currencies: ",
			wantNotContain: "no brokers",
			wantPrefix:     "currencies: ",
		},
		{
			name: "kafka before topic",
			mutate: func(cfg *Config) {
				cfg.Kafka.Brokers = nil
				cfg.Kafka.Topic = ""
			},
			wantContains:   "no brokers",
			wantNotContain: "currencies: ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)

			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantContains)
			assert.NotContains(t, err.Error(), tt.wantNotContain)
			assert.True(t, strings.HasPrefix(err.Error(), tt.wantPrefix), "error %q must start with %q", err.Error(), tt.wantPrefix)
			assert.Len(t, cfg.CurrencySet.List(), 0)
		})
	}
}
