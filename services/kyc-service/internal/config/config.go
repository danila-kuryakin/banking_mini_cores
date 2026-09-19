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

type Config struct {
	Server         config.Server         `mapstructure:"server"`
	Postgres       config.DataBaseConfig `mapstructure:"database"`
	Upstreams      Upstreams             `mapstructure:"upstreams"`
	AutoChecks     AutoChecks            `mapstructure:"auto_checks"`
	RequestTimeout time.Duration         `mapstructure:"request_timeout"`
}

func (c *Config) Validate() error {
	if c.Upstreams.Customer == "" {
		return errors.New("upstream address for customer-service is empty")
	}

	if c.Upstreams.Filestore == "" {
		return errors.New("upstream address for filestore-service is empty")
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

	return cfg, nil
}
