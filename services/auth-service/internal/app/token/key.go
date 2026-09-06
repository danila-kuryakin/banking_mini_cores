package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
)

func privateKey(path string, logger *slog.Logger) (*rsa.PrivateKey, error) {
	if path == "" {
		logger.Warn("jwt private key path is not set, a temporary key is generated for this run; " +
			"tokens issued now will stop verifying after a restart")

		return generatePrivateKey()
	}

	key, err := loadPrivateKey(path)
	if err == nil {
		return key, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	logger.Warn("jwt private key not found, generating and saving a new one", "path", path)

	key, err = generatePrivateKey()
	if err != nil {
		return nil, err
	}

	return key, savePrivateKey(path, key)
}

func savePrivateKey(path string, key *rsa.PrivateKey) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}

	encoded := pem.EncodeToMemory(&pem.Block{
		Type:  domain.PEM_RSA_PRIVATE_KEY,
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return fmt.Errorf("write private key %s: %w", path, err)
	}

	return nil
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
