package config

import (
	"errors"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
)

type JWT struct {
	KeysDir    string        `mapstructure:"keys_dir"`
	AccessTTL  time.Duration `mapstructure:"access_ttl"`
	RefreshTTL time.Duration `mapstructure:"refresh_ttl"`
}

type Admin struct {
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
}

type Upstreams struct {
	Customer string `mapstructure:"customer"`
}

type Config struct {
	Server         config.Server         `mapstructure:"server"`
	JWKS           config.Server         `mapstructure:"jwks"`
	Postgres       config.DataBaseConfig `mapstructure:"database"`
	JWT            JWT                   `mapstructure:"jwt"`
	Admin          Admin                 `mapstructure:"admin"`
	Upstreams      Upstreams             `mapstructure:"upstreams"`
	RequestTimeout time.Duration         `mapstructure:"request_timeout"`
}

// Validate вызывает config.Read сам, до подстановки умолчаний в Load. Поэтому
// проверяем здесь только то, у чего умолчания быть не может.
func (c *Config) Validate() error {
	if c.Upstreams.Customer == "" {
		return errors.New("upstream address for customer-service is empty")
	}

	return nil
}

func Load() (*Config, error) {
	cfg, err := config.Read[Config]("./configs")
	if err != nil {
		return nil, err
	}

	if cfg.JWT.KeysDir == "" {
		cfg.JWT.KeysDir = domain.DEFAULT_JWT_KEYS_DIR
	}

	if cfg.JWT.AccessTTL <= 0 {
		cfg.JWT.AccessTTL = domain.DEFAULT_ACCESS_TTL
	}

	if cfg.JWT.RefreshTTL <= 0 {
		cfg.JWT.RefreshTTL = domain.DEFAULT_REFRESH_TTL
	}

	if cfg.Admin.Email == "" {
		cfg.Admin.Email = domain.DEFAULT_ADMIN_EMAIL
	}

	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = domain.DEFAULT_REQUEST_TIMEOUT
	}

	return cfg, nil
}
