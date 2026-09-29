package config

import (
	"errors"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
)

// Consumer - повторы обработки события до ухода в DLQ.
type Consumer struct {
	MaxAttempts int           `mapstructure:"max_attempts"`
	MinBackoff  time.Duration `mapstructure:"min_backoff"`
	MaxBackoff  time.Duration `mapstructure:"max_backoff"`
}

type Config struct {
	Server         config.Server         `mapstructure:"server"`
	Postgres       config.DataBaseConfig `mapstructure:"database"`
	Kafka          config.KafkaConfig    `mapstructure:"kafka"`
	Consumer       Consumer              `mapstructure:"consumer"`
	RequestTimeout time.Duration         `mapstructure:"request_timeout"`
}

// Validate: kafka.topics - что читаем, kafka.topic - куда откладываем непрочитанное (DLQ).
func (c *Config) Validate() error {
	if err := c.Kafka.Validate(); err != nil {
		return err
	}

	if len(c.Kafka.TopicList()) == 0 {
		return errors.New("kafka: topics to read are empty")
	}

	if c.Kafka.GroupID == "" {
		return errors.New("kafka: group_id is empty")
	}

	if c.Kafka.Topic == "" {
		return errors.New("kafka: dlq topic is empty")
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

	if cfg.Consumer.MaxAttempts <= 0 {
		cfg.Consumer.MaxAttempts = domain.DEFAULT_RETRY_MAX_ATTEMPTS
	}

	if cfg.Consumer.MinBackoff <= 0 {
		cfg.Consumer.MinBackoff = domain.DEFAULT_RETRY_MIN_BACKOFF
	}

	if cfg.Consumer.MaxBackoff <= 0 {
		cfg.Consumer.MaxBackoff = domain.DEFAULT_RETRY_MAX_BACKOFF
	}

	return cfg, nil
}
