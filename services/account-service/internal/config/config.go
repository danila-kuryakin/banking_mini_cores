package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
)

type Upstreams struct {
	Customer string `mapstructure:"customer"`
	Ledger   string `mapstructure:"ledger"`
}

type AccountNumber struct {
	BalancePrefix string `mapstructure:"balance_prefix"`
}

type Config struct {
	Server          config.Server         `mapstructure:"server"`
	Postgres        config.DataBaseConfig `mapstructure:"database"`
	Currencies      []string              `mapstructure:"currencies"`
	DefaultCurrency string                `mapstructure:"default_currency"`
	AccountNumber   AccountNumber         `mapstructure:"account_number"`
	Upstreams       Upstreams             `mapstructure:"upstreams"`
	RequestTimeout  time.Duration         `mapstructure:"request_timeout"`

	CurrencySet         models.CurrencySet      `mapstructure:"-"`
	DefaultCurrencyCode models.Currency         `mapstructure:"-"`
	NumberGenerator     *models.NumberGenerator `mapstructure:"-"`
}

func (c *Config) Validate() error {
	if c.Upstreams.Customer == "" {
		return errors.New("upstream address for customer-service is empty")
	}

	if c.Upstreams.Ledger == "" {
		return errors.New("upstream address for ledger-service is empty")
	}

	currencies, err := models.NewCurrencySet(c.Currencies)
	if err != nil {
		return fmt.Errorf("currencies: %w", err)
	}

	defaultCurrency, err := models.ParseCurrency(c.DefaultCurrency)
	if err != nil {
		return fmt.Errorf("default_currency: %w", err)
	}

	if !currencies.Contains(defaultCurrency) {
		return fmt.Errorf("default_currency %s is not in currencies", defaultCurrency)
	}

	numbers, err := models.NewNumberGenerator(c.AccountNumber.BalancePrefix)
	if err != nil {
		return fmt.Errorf("account_number.balance_prefix: %w", err)
	}

	c.CurrencySet = currencies
	c.DefaultCurrencyCode = defaultCurrency
	c.NumberGenerator = numbers

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

	return cfg, nil
}
