package config

import (
	"fmt"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
)

// Upstreams are the gRPC addresses of the services the gateway forwards to.
type Upstreams struct {
	Auth         string
	Customer     string
	KYC          string
	Document     string
	Account      string
	Ledger       string
	Antifraud    string
	Notification string
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

type Config struct {
	Base           config.Base
	HTTPAddr       string
	GatewayAddr    string // Для health checks. Остальные микросервисы доступны по тому же адресу Upstreams
	Upstreams      Upstreams
	JWKSURL        string
	CORSOrigins    []string
	RequestTimeout time.Duration
}

func Load() (Config, error) {
	base, err := config.LoadBase("api-gateway", 0)
	if err != nil {
		return Config{}, err
	}

	requestTimeout, err := config.Duration("REQUEST_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Base:        base,
		HTTPAddr:    config.String("HTTP_ADDR", ":8080"),
		GatewayAddr: config.String("GATEWAY_ADDR", ":50059"),
		Upstreams: Upstreams{
			Auth:         config.String("AUTH_SERVICE_ADDR", "localhost:50051"),
			Customer:     config.String("CUSTOMER_SERVICE_ADDR", "customer-service:50052"),
			KYC:          config.String("KYC_SERVICE_ADDR", "kyc-service:50053"),
			Document:     config.String("DOCUMENT_SERVICE_ADDR", "document-service:50054"),
			Account:      config.String("ACCOUNT_SERVICE_ADDR", "account-service:50055"),
			Ledger:       config.String("LEDGER_SERVICE_ADDR", "ledger-service:50056"),
			Antifraud:    config.String("ANTIFRAUD_SERVICE_ADDR", "antifraud-service:50057"),
			Notification: config.String("NOTIFICATION_SERVICE_ADDR", "notification-service:50058"),
		},
		JWKSURL:        config.String("JWKS_URL", ""),
		CORSOrigins:    config.StringSlice("CORS_ORIGINS", []string{"http://localhost:3000"}),
		RequestTimeout: requestTimeout,
	}

	for name, addr := range cfg.Upstreams.All() {
		if addr == "" {
			return Config{}, fmt.Errorf("config: upstream address for %s is empty", name)
		}
	}

	return cfg, nil
}
