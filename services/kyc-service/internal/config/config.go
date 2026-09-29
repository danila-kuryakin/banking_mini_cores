package config

import (
	"errors"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
)

type Upstreams struct {
	Customer  string `mapstructure:"customer"`
	Filestore string `mapstructure:"filestore"`
}

// AutoChecks - настройки фонового разбора заявок.
type AutoChecks struct {
	Interval  time.Duration `mapstructure:"interval"`
	BatchSize int           `mapstructure:"batch_size"`
}

// Outbox - настройки публикации событий заявок в Kafka.
type Outbox struct {
	Interval  time.Duration `mapstructure:"interval"`
	BatchSize int           `mapstructure:"batch_size"`
}

type Config struct {
	Server         config.Server         `mapstructure:"server"`
	Postgres       config.DataBaseConfig `mapstructure:"database"`
	Kafka          config.KafkaConfig    `mapstructure:"kafka"`
	Upstreams      Upstreams             `mapstructure:"upstreams"`
	AutoChecks     AutoChecks            `mapstructure:"auto_checks"`
	Outbox         Outbox                `mapstructure:"outbox"`
	RequestTimeout time.Duration         `mapstructure:"request_timeout"`
}

func (c *Config) Validate() error {
	if c.Upstreams.Customer == "" {
		return errors.New("upstream address for customer-service is empty")
	}

	if c.Upstreams.Filestore == "" {
		return errors.New("upstream address for filestore-service is empty")
	}

	if err := c.Kafka.Validate(); err != nil {
		return err
	}

	if c.Kafka.Topic == "" {
		return errors.New("kafka: topic for kyc events is empty")
	}

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

	if cfg.AutoChecks.Interval <= 0 {
		cfg.AutoChecks.Interval = domain.DEFAULT_AUTO_CHECKS_INTERVAL
	}

	if cfg.AutoChecks.BatchSize <= 0 {
		cfg.AutoChecks.BatchSize = domain.DEFAULT_AUTO_CHECKS_BATCH_SIZE
	}

	if cfg.Outbox.Interval <= 0 {
		cfg.Outbox.Interval = domain.DEFAULT_OUTBOX_INTERVAL
	}

	if cfg.Outbox.BatchSize <= 0 {
		cfg.Outbox.BatchSize = domain.DEFAULT_OUTBOX_BATCH_SIZE
	}

	return cfg, nil
}
