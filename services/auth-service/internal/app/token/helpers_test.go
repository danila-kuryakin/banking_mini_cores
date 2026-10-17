package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	TEST_KEY_FILE_A  = "20250101T000000.pem"
	TEST_KEY_FILE_B  = "20250201T000000.pem"
	TEST_ACCESS_TTL  = 15 * time.Minute
	TEST_REFRESH_TTL = 24 * time.Hour
	TEST_KEYS_COUNT  = 3
)

// Кеш RSA-ключей: генерация 2048 бит медленная, создаём один раз на пакет.
var (
	testKeysOnce sync.Once
	testKeys     []*rsa.PrivateKey
	testKeysErr  error
)

// testKey возвращает закешированный ключ по индексу.
func testKey(t *testing.T, index int) *rsa.PrivateKey {
	t.Helper()
	testKeysOnce.Do(func() {
		for range TEST_KEYS_COUNT {
			key, err := rsa.GenerateKey(rand.Reader, domain.RSA_KEY_BITS)
			if err != nil {
				testKeysErr = err
				return
			}
			testKeys = append(testKeys, key)
		}
	})
	require.NoError(t, testKeysErr)
	require.Less(t, index, len(testKeys), "test key index out of range")
	return testKeys[index]
}

// testLogger отдаёт логгер, который ничего не пишет.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.DiscardHandler)
}

// writePEM кладёт PEM-блок с заданными байтами в каталог.
func writePEM(t *testing.T, dir, name, blockType string, der []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}

// writePKCS1Key кладёт RSA-ключ в формате PKCS#1.
func writePKCS1Key(t *testing.T, dir, name string, key *rsa.PrivateKey) string {
	t.Helper()
	return writePEM(t, dir, name, domain.PEM_RSA_PRIVATE_KEY, x509.MarshalPKCS1PrivateKey(key))
}

// newTestManager создаёт менеджер над каталогом с заданными ключами.
func newTestManager(t *testing.T, accessTTL time.Duration, files map[string]*rsa.PrivateKey) *Manager {
	t.Helper()
	dir := t.TempDir()
	for name, key := range files {
		writePKCS1Key(t, dir, name, key)
	}
	manager, err := NewManager(dir, accessTTL, TEST_REFRESH_TTL, testLogger(t))
	require.NoError(t, err)
	return manager
}

// testUser отдаёт клиента со свежим идентификатором.
func testUser(t *testing.T) *models.User {
	t.Helper()
	return &models.User{ID: uuid.New(), Email: "user@example.com", Role: domain.ROLE_CLIENT}
}

// headerKid достаёт kid из заголовка токена без проверки подписи.
func headerKid(t *testing.T, raw string) string {
	t.Helper()
	parsed, _, err := jwt.NewParser().ParseUnverified(raw, jwt.MapClaims{})
	require.NoError(t, err)
	kid, _ := parsed.Header[domain.JWK_KEY_ID_HEADER].(string)
	return kid
}

// jwksKids собирает kid всех ключей из JWKS.
func jwksKids(t *testing.T, set Set) []string {
	t.Helper()
	kids := make([]string, 0, len(set.Keys))
	for _, key := range set.Keys {
		kids = append(kids, key.KeyID)
	}
	return kids
}

// signRS256 подписывает claims ключом с заданным kid; пустой kid не ставится.
func signRS256(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if kid != "" {
		token.Header[domain.JWK_KEY_ID_HEADER] = kid
	}
	raw, err := token.SignedString(key)
	require.NoError(t, err)
	return raw
}

// validClaims отдаёт корректный набор claims для токена «руками».
func validClaims(t *testing.T) jwt.MapClaims {
	t.Helper()
	now := time.Now()
	return jwt.MapClaims{
		"sub":  uuid.NewString(),
		"iat":  now.Unix(),
		"exp":  now.Add(TEST_ACCESS_TTL).Unix(),
		"role": string(domain.ROLE_CLIENT),
	}
}
