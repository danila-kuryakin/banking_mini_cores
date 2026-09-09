package config

import (
	"fmt"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
)

// Upstreams are the gRPC addresses of the services the gateway forwards to.
type Upstreams struct {
	Auth         string `mapstructure:"auth"`
	Customer     string `mapstructure:"customer"`
	KYC          string `mapstructure:"kyc"`
	Filestore    string `mapstructure:"filestore"`
	Account      string `mapstructure:"account"`
	Ledger       string `mapstructure:"ledger"`
	Antifraud    string `mapstructure:"antifraud"`
	Notification string `mapstructure:"notification"`
}

func (u Upstreams) All() map[string]string {
	return map[string]string{
		"auth-service":         u.Auth,
		"customer-service":     u.Customer,
		"kyc-service":          u.KYC,
		"filestore-service":    u.Filestore,
		"account-service":      u.Account,
		"ledger-service":       u.Ledger,
		"antifraud-service":    u.Antifraud,
		"notification-service": u.Notification,
	}
}

type Config struct {
	RestServer          config.Server `mapstructure:"server"`
	GRPCServer          config.Server `mapstructure:"grpc_client"`
	Upstreams           Upstreams     `mapstructure:"upstreams"`
	JWKSURL             string        `mapstructure:"jwks_url"`
	JWKSRefreshInterval time.Duration `mapstructure:"jwks_refresh_interval"`
	RequestTimeout      time.Duration `mapstructure:"request_timeout"`
	ShutdownTimeout     time.Duration `mapstructure:"shutdown_timeout"`
}

func (c *Config) Validate() error {
	for name, addr := range c.Upstreams.All() {
		if addr == "" {
			return fmt.Errorf("upstream address for %s is empty", name)
		}
	}

	return nil
}

func Load() (*Config, error) {
	cfg, err := config.Read[Config]("./configs")
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = domain.DEFAULT_REQUEST_TIMEOUT
	}

	if cfg.JWKSRefreshInterval <= 0 {
		cfg.JWKSRefreshInterval = domain.DEFAULT_JWKS_REFRESH_INTERVAL
	}

	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = domain.DEFAULT_SHUTDOWN_TIMEOUT
	}

	return cfg, nil
}
