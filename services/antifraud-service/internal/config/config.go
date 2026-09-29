package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

// Consumer - паузы между повторами обработки события.
type Consumer struct {
	MinBackoff time.Duration `mapstructure:"min_backoff"`
	MaxBackoff time.Duration `mapstructure:"max_backoff"`
}

type Config struct {
	Server         config.Server         `mapstructure:"server"`
	Postgres       config.DataBaseConfig `mapstructure:"database"`
	Kafka          config.KafkaConfig    `mapstructure:"kafka"`
	Currencies     []string              `mapstructure:"currencies"`
	Consumer       Consumer              `mapstructure:"consumer"`
	RequestTimeout time.Duration         `mapstructure:"request_timeout"`

	CurrencySet models.CurrencySet `mapstructure:"-"`
}

// Validate проверяет конфиг и сохраняет разобранный список валют; вызывается из config.Read.
func (c *Config) Validate() error {
	currencies, err := models.NewCurrencySet(c.Currencies)
	if err != nil {
		return fmt.Errorf("currencies: %w", err)
	}

	if err := c.Kafka.Validate(); err != nil {
		return err
	}

	if len(c.Kafka.TopicList()) == 0 {
		return errors.New("kafka: topics to read are empty")
	}

	if c.Kafka.GroupID == "" {
		return errors.New("kafka: group_id is empty")
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

	if cfg.Consumer.MinBackoff <= 0 {
		cfg.Consumer.MinBackoff = domain.DEFAULT_RETRY_MIN_BACKOFF
	}

	if cfg.Consumer.MaxBackoff <= 0 {
		cfg.Consumer.MaxBackoff = domain.DEFAULT_RETRY_MAX_BACKOFF
	}

	return cfg, nil
}
