package config

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

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

type Login struct {
	MaxFailedAttempts int           `mapstructure:"max_failed_attempts"`
	Window            time.Duration `mapstructure:"window"`
	LockDuration      time.Duration `mapstructure:"lock_duration"`
}

type PasswordHashing struct {
	MaxConcurrent int           `mapstructure:"max_concurrent"`
	MaxWait       time.Duration `mapstructure:"max_wait"`
}

type Upstreams struct {
	Customer string `mapstructure:"customer"`
}

type Config struct {
	Server          config.Server         `mapstructure:"server"`
	JWKS            config.Server         `mapstructure:"jwks"`
	Postgres        config.DataBaseConfig `mapstructure:"database"`
	JWT             JWT                   `mapstructure:"jwt"`
	Admin           Admin                 `mapstructure:"admin"`
	Upstreams       Upstreams             `mapstructure:"upstreams"`
	Login           Login                 `mapstructure:"login"`
	PasswordHashing PasswordHashing       `mapstructure:"password_hashing"`
	RequestTimeout  time.Duration         `mapstructure:"request_timeout"`
}

// Validate вызывает config.Read сам, до подстановки умолчаний в Load. Поэтому
// проверяем здесь только то, у чего умолчания быть не может.
func (c *Config) Validate() error {
	if c.Upstreams.Customer == "" {
		return errors.New("upstream address for customer-service is empty")
	}

	if c.Login.MaxFailedAttempts < 0 {
		return errors.New("login.max_failed_attempts must not be negative")
	}

	if c.Login.Window < 0 {
		return errors.New("login.window must not be negative")
	}

	if c.Login.LockDuration < 0 {
		return errors.New("login.lock_duration must not be negative")
	}

	if c.PasswordHashing.MaxConcurrent < 0 {
		return errors.New("password_hashing.max_concurrent must not be negative")
	}

	if c.PasswordHashing.MaxWait < 0 {
		return errors.New("password_hashing.max_wait must not be negative")
	}

	if c.Admin.Password != "" {
		length := utf8.RuneCountInString(c.Admin.Password)
		if length < domain.PASSWORD_MIN_RUNES || length > domain.PASSWORD_MAX_RUNES {
			return fmt.Errorf("admin.password must be from %d to %d characters long", domain.PASSWORD_MIN_RUNES, domain.PASSWORD_MAX_RUNES)
		}
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

	if cfg.Login.MaxFailedAttempts <= 0 {
		cfg.Login.MaxFailedAttempts = domain.DEFAULT_LOGIN_MAX_FAILED_ATTEMPTS
	}

	if cfg.Login.Window <= 0 {
		cfg.Login.Window = domain.DEFAULT_LOGIN_WINDOW
	}

	if cfg.Login.LockDuration <= 0 {
		cfg.Login.LockDuration = domain.DEFAULT_LOGIN_LOCK_DURATION
	}

	if cfg.PasswordHashing.MaxConcurrent <= 0 {
		cfg.PasswordHashing.MaxConcurrent = domain.DEFAULT_PASSWORD_HASHING_MAX_CONCURRENT
	}

	if cfg.PasswordHashing.MaxWait <= 0 {
		cfg.PasswordHashing.MaxWait = domain.DEFAULT_PASSWORD_HASHING_MAX_WAIT
	}

	return cfg, nil
}
