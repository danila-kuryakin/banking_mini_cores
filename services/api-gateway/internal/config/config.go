package config

import (
	"fmt"
	"log"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
)

// Upstreams are the gRPC addresses of the services the gateway forwards to.
type Upstreams struct {
	Auth         string `mapstructure:"auth"`
	Customer     string `mapstructure:"customer"`
	KYC          string `mapstructure:"kyc"`
	Document     string `mapstructure:"document"`
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
		"document-service":     u.Document,
		"account-service":      u.Account,
		"ledger-service":       u.Ledger,
		"antifraud-service":    u.Antifraud,
		"notification-service": u.Notification,
	}
}

// Config is the api-gateway configuration beyond the common base.
type Config struct {
	Server config.Server `mapstructure:"server"`
	// GRPC - адрес health-сервера. Остальные микросервисы доступны по адресам
	// из Upstreams.
	GRPC            config.Server `mapstructure:"grpc"`
	Upstreams       Upstreams     `mapstructure:"upstreams"`
	JWKSURL         string        `mapstructure:"jwks_url"`
	RequestTimeout  time.Duration `mapstructure:"request_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// Validate реализует config.Validator: Read вызовет его сам после разбора.
// Пустой адрес апстрима - это отказ на первом же запросе к сервису, поэтому
// ловим его на старте.
func (c *Config) Validate() error {
	for name, addr := range c.Upstreams.All() {
		if addr == "" {
			return fmt.Errorf("upstream address for %s is empty", name)
		}
	}

	return nil
}

// Load reads the configuration from the environment.
func Load() (*Config, error) {

	cfg, err := config.Read[Config]("./configs")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	return cfg, nil
}
