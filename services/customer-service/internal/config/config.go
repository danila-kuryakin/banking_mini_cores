package config

import (
	"log"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
)

type Config struct {
	Server         config.Server         `mapstructure:"server"`
	Postgres       config.DataBaseConfig `mapstructure:"database"`
	RequestTimeout time.Duration         `mapstructure:"request_timeout"`
}

func Load() (*Config, error) {

	cfg, err := config.Read[Config]("./configs")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = domain.DEFAULT_REQEST_TIMEOUT
	}

	return cfg, nil
}
