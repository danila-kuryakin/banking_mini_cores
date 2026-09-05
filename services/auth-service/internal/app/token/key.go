package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
)

func privateKey(path string, logger *slog.Logger) (*rsa.PrivateKey, error) {
	if path == "" {
		logger.Warn("jwt private key path is not set, a temporary key is generated for this run")

		return generatePrivateKey()
	}

	return loadPrivateKey(path)
}

func generatePrivateKey() (*rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, domain.RSA_KEY_BITS)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}

	return key, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key %s: %w", path, err)
	}

	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, domain.ErrPrivateKeyNotPEM
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key %s: %w", path, err)
	}

	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, domain.ErrPrivateKeyNotRSA
	}

	return key, nil
}
