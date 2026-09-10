package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
)

// keyFile - приватный ключ вместе с именем файла, из которого он прочитан.
type keyFile struct {
	name string
	key  *rsa.PrivateKey
}

// loadKeys читает каталог ключей и возвращает их в порядке имён файлов.
// Последний в этом порядке становится активным - им подписываются новые токены,
// остальные остаются в наборе и продолжают проверять уже выданные.
//
// Порядок задаёт имя файла (метка времени), а не mtime: mtime сбивается при
// копировании, распаковке и checkout, имя - нет.
func loadKeys(dir string, logger *slog.Logger) ([]keyFile, error) {
	if dir == "" {
		logger.Warn("jwt keys dir is not set, a temporary key is generated for this run; " +
			"tokens issued now will stop verifying after a restart")

		key, err := generatePrivateKey()
		if err != nil {
			return nil, err
		}

		return []keyFile{{name: "(memory)", key: key}}, nil
	}

	names, err := keyFileNames(dir)
	if err != nil {
		return nil, err
	}

	// Каталога нет или он пуст - первый запуск: заводим ключ, чтобы сервис
	// поднялся сам, без ручной подготовки.
	if len(names) == 0 {
		logger.Warn("jwt keys dir has no keys, generating the first one", "dir", dir)

		name, key, err := generateAndSaveKey(dir)
		if err != nil {
			return nil, err
		}

		return []keyFile{{name: name, key: key}}, nil
	}

	keys := make([]keyFile, 0, len(names))
	for _, name := range names {
		key, err := loadPrivateKey(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}

		keys = append(keys, keyFile{name: name, key: key})
	}

	return keys, nil
}

func keyFileNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("read jwt keys dir %s: %w", dir, err)
	}

	var names []string

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != domain.JWT_KEY_FILE_EXT {
			continue
		}

		names = append(names, e.Name())
	}

	sort.Strings(names)

	return names, nil
}

func generateAndSaveKey(dir string) (string, *rsa.PrivateKey, error) {
	key, err := generatePrivateKey()
	if err != nil {
		return "", nil, err
	}

	name := time.Now().UTC().Format(domain.JWT_KEY_NAME_LAYOUT) + domain.JWT_KEY_FILE_EXT

	if err := savePrivateKey(filepath.Join(dir, name), key); err != nil {
		return "", nil, err
	}

	return name, key, nil
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
		return nil, fmt.Errorf("%s: %w", path, domain.ErrPrivateKeyNotPEM)
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
		return nil, fmt.Errorf("%s: %w", path, domain.ErrPrivateKeyNotRSA)
	}

	return key, nil
}
