package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

type Upstreams struct {
	Account   string `mapstructure:"account"`
	Antifraud string `mapstructure:"antifraud"`
}

type Outbox struct {
	Interval  time.Duration `mapstructure:"interval"`
	BatchSize int           `mapstructure:"batch_size"`
}

type Config struct {
	Server         config.Server         `mapstructure:"server"`
	Postgres       config.DataBaseConfig `mapstructure:"database"`
	Kafka          config.KafkaConfig    `mapstructure:"kafka"`
	Currencies     []string              `mapstructure:"currencies"`
	Upstreams      Upstreams             `mapstructure:"upstreams"`
	Outbox         Outbox                `mapstructure:"outbox"`
	RequestTimeout time.Duration         `mapstructure:"request_timeout"`

	CurrencySet models.CurrencySet `mapstructure:"-"`
}

// Validate проверяет конфиг и сохраняет разобранный список валют; вызывается из config.Read.
func (c *Config) Validate() error {
	if c.Upstreams.Account == "" {
		return errors.New("upstream address for account-service is empty")
	}

	if c.Upstreams.Antifraud == "" {
		return errors.New("upstream address for antifraud-service is empty")
	}

	currencies, err := models.NewCurrencySet(c.Currencies)
	if err != nil {
		return fmt.Errorf("currencies: %w", err)
	}

	if err := c.Kafka.Validate(); err != nil {
		return err
	}

	if c.Kafka.Topic == "" {
		return errors.New("kafka: topic for ledger events is empty")
	}

	c.CurrencySet = currencies

	return nil
}

func Load() (*Config, error) {
	cfg, err := config.Read[Config]("./configs")
	if err != nil {
		return nil, err
	}

	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = domain.DEFAULT_REQUEST_TIMEOUT
	}

	if cfg.Outbox.Interval <= 0 {
		cfg.Outbox.Interval = domain.DEFAULT_OUTBOX_INTERVAL
	}

	if cfg.Outbox.BatchSize <= 0 {
		cfg.Outbox.BatchSize = domain.DEFAULT_OUTBOX_BATCH_SIZE
	}

	return cfg, nil
}
