package config

import (
	"log"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
)

// Config is the auth-service configuration beyond the common base.
type Config struct {
	Server   config.Server         `mapstructure:"server"`
	Postgres config.DataBaseConfig `mapstructure:"database"`
	Kafka    config.KafkaConfig    `mapstructure:"kafka"`
}

// Load reads the configuration from the environment.
func Load() (*Config, error) {

	cfg, err := config.Read[Config]("./configs")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	return cfg, nil
}
