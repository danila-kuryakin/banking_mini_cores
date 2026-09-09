package config

import (
	"fmt"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
)

type MinIOConfig struct {
	Endpoint        string        `mapstructure:"endpoint"`
	AccessKey       string        `mapstructure:"access_key"`
	SecretKey       string        `mapstructure:"secret_key"`
	BucketName      string        `mapstructure:"bucket_name"`
	UseSSL          bool          `mapstructure:"use_ssl"`
	ConnectTimeout  time.Duration `mapstructure:"connect_timeout"`
	HealthCheckFreq time.Duration `mapstructure:"health_check_freq"`
}

type Config struct {
	Server         config.Server         `mapstructure:"server"`
	Postgres       config.DataBaseConfig `mapstructure:"database"`
	MinIOConfig    MinIOConfig           `mapstructure:"minio_config"`
	RequestTimeout time.Duration         `mapstructure:"request_timeout"`
}

func Load() (*Config, error) {
	cfg, err := config.Read[Config]("./configs")
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	cfg.setDefaults()

	return cfg, nil
}

func (c *Config) setDefaults() {
	if c.RequestTimeout <= 0 {
		c.RequestTimeout = domain.DEFAULT_REQUEST_TIMEOUT
	}

	if c.MinIOConfig.ConnectTimeout <= 0 {
		c.MinIOConfig.ConnectTimeout = domain.DEFAULT_MINIO_CONNECT_TIMEOUT
	}

	if c.MinIOConfig.HealthCheckFreq <= 0 {
		c.MinIOConfig.HealthCheckFreq = domain.DEFAULT_MINIO_HEALTH_CHECK_FREQ
	}
}

func (c *Config) Validate() error {
	required := []struct {
		key   string
		value string
	}{
		{"server.port", c.Server.Port},
		{"minio_config.endpoint", c.MinIOConfig.Endpoint},
		{"minio_config.access_key", c.MinIOConfig.AccessKey},
		{"minio_config.secret_key", c.MinIOConfig.SecretKey},
		{"minio_config.bucket_name", c.MinIOConfig.BucketName},
	}

	for _, field := range required {
		if field.value == "" {
			return fmt.Errorf("%w: %s", domain.ErrConfigValueRequired, field.key)
		}
	}

	return nil
}
