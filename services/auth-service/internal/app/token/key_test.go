package token

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const TEST_KEY_FILE_PATTERN = `^\d{8}T\d{6}\.pem$`

// listDir отдаёт имена записей каталога.
func listDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func TestNewManager_EmptyKeysDir(t *testing.T) {
	manager, err := NewManager("", TEST_ACCESS_TTL, TEST_REFRESH_TTL, testLogger(t))
	require.NoError(t, err)

	assert.Len(t, manager.JWKS().Keys, 1)

	user := testUser(t)
	pair, err := manager.NewPair(user)
	require.NoError(t, err)
	claims, err := manager.ParseAccess(pair.Access)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
}

func TestNewManager_MissingDirCreatesKey(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "keys")

	_, err := NewManager(dir, TEST_ACCESS_TTL, TEST_REFRESH_TTL, testLogger(t))
	require.NoError(t, err)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	require.True(t, info.IsDir())

	names := listDir(t, dir)
	require.Len(t, names, 1)
	assert.Regexp(t, regexp.MustCompile(TEST_KEY_FILE_PATTERN), names[0])
}

func TestNewManager_EmptyDirCreatesKey(t *testing.T) {
	dir := t.TempDir()

	_, err := NewManager(dir, TEST_ACCESS_TTL, TEST_REFRESH_TTL, testLogger(t))
	require.NoError(t, err)

	names := listDir(t, dir)
	require.Len(t, names, 1)
	assert.Regexp(t, regexp.MustCompile(TEST_KEY_FILE_PATTERN), names[0])
}

func TestNewManager_LoadsExistingKeys(t *testing.T) {
	keyA, keyB := testKey(t, 0), testKey(t, 1)
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{
		TEST_KEY_FILE_A: keyA,
		TEST_KEY_FILE_B: keyB,
	})

	kids := jwksKids(t, manager.JWKS())
	assert.Len(t, kids, 2)
	assert.Contains(t, kids, keyID(&keyA.PublicKey))
	assert.Contains(t, kids, keyID(&keyB.PublicKey))

	pair, err := manager.NewPair(testUser(t))
	require.NoError(t, err)
	assert.Equal(t, keyID(&keyB.PublicKey), headerKid(t, pair.Access))
}

func TestNewManager_ActiveKeyByNameNotMtime(t *testing.T) {
	keyA, keyB := testKey(t, 0), testKey(t, 1)
	dir := t.TempDir()
	pathA := writePKCS1Key(t, dir, TEST_KEY_FILE_A, keyA)
	writePKCS1Key(t, dir, TEST_KEY_FILE_B, keyB)
	future := time.Now().Add(24 * time.Hour)
	require.NoError(t, os.Chtimes(pathA, future, future))

	manager, err := NewManager(dir, TEST_ACCESS_TTL, TEST_REFRESH_TTL, testLogger(t))
	require.NoError(t, err)

	pair, err := manager.NewPair(testUser(t))
	require.NoError(t, err)
	assert.Equal(t, keyID(&keyB.PublicKey), headerKid(t, pair.Access))
}

func TestNewManager_IgnoresForeignFiles(t *testing.T) {
	key := testKey(t, 0)
	dir := t.TempDir()
	writePKCS1Key(t, dir, TEST_KEY_FILE_A, key)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("garbage"), 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "x.pem"), 0o700))

	manager, err := NewManager(dir, TEST_ACCESS_TTL, TEST_REFRESH_TTL, testLogger(t))
	require.NoError(t, err)

	kids := jwksKids(t, manager.JWKS())
	require.Len(t, kids, 1)
	assert.Equal(t, keyID(&key.PublicKey), kids[0])
}

func TestNewManager_NotPEM(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, TEST_KEY_FILE_A), []byte("not a pem file"), 0o600))

	_, err := NewManager(dir, TEST_ACCESS_TTL, TEST_REFRESH_TTL, testLogger(t))

	require.ErrorIs(t, err, domain.ErrPrivateKeyNotPEM)
}

func TestNewManager_NotRSA(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(ecKey)
	require.NoError(t, err)
	dir := t.TempDir()
	writePEM(t, dir, TEST_KEY_FILE_A, "PRIVATE KEY", der)

	_, err = NewManager(dir, TEST_ACCESS_TTL, TEST_REFRESH_TTL, testLogger(t))

	require.ErrorIs(t, err, domain.ErrPrivateKeyNotRSA)
}

func TestNewManager_PKCS8RSA(t *testing.T) {
	key := testKey(t, 0)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	dir := t.TempDir()
	writePEM(t, dir, TEST_KEY_FILE_A, "PRIVATE KEY", der)

	manager, err := NewManager(dir, TEST_ACCESS_TTL, TEST_REFRESH_TTL, testLogger(t))
	require.NoError(t, err)

	kids := jwksKids(t, manager.JWKS())
	require.Len(t, kids, 1)
	assert.Equal(t, keyID(&key.PublicKey), kids[0])
}

func TestNewManager_BrokenDER(t *testing.T) {
	dir := t.TempDir()
	writePEM(t, dir, TEST_KEY_FILE_A, domain.PEM_RSA_PRIVATE_KEY, []byte("definitely not der"))

	_, err := NewManager(dir, TEST_ACCESS_TTL, TEST_REFRESH_TTL, testLogger(t))

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrPrivateKeyNotPEM)
	assert.NotErrorIs(t, err, domain.ErrPrivateKeyNotRSA)
}

// generatedKeyFile создаёт менеджер на пустом каталоге и отдаёт путь к единственному ключу.
func generatedKeyFile(t *testing.T) (*Manager, string) {
	t.Helper()
	dir := t.TempDir()
	manager, err := NewManager(dir, TEST_ACCESS_TTL, TEST_REFRESH_TTL, testLogger(t))
	require.NoError(t, err)
	names := listDir(t, dir)
	require.Len(t, names, 1)
	return manager, filepath.Join(dir, names[0])
}

func TestNewManager_GeneratedKeyPersists(t *testing.T) {
	dir := t.TempDir()
	first, err := NewManager(dir, TEST_ACCESS_TTL, TEST_REFRESH_TTL, testLogger(t))
	require.NoError(t, err)
	second, err := NewManager(dir, TEST_ACCESS_TTL, TEST_REFRESH_TTL, testLogger(t))
	require.NoError(t, err)

	assert.Len(t, listDir(t, dir), 1)
	assert.Equal(t, jwksKids(t, first.JWKS()), jwksKids(t, second.JWKS()))

	user := testUser(t)
	pair, err := first.NewPair(user)
	require.NoError(t, err)
	claims, err := second.ParseAccess(pair.Access)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
}

func TestNewManager_GeneratedKeyFormat(t *testing.T) {
	manager, path := generatedKeyFile(t)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	block, _ := pem.Decode(data)
	require.NotNil(t, block, "generated key file is not PEM")
	assert.Equal(t, domain.PEM_RSA_PRIVATE_KEY, block.Type)
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)

	kids := jwksKids(t, manager.JWKS())
	require.Len(t, kids, 1)
	assert.Equal(t, keyID(&key.PublicKey), kids[0])
}

func TestNewManager_GeneratedKeyPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file permissions only")
	}
	_, path := generatedKeyFile(t)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}
