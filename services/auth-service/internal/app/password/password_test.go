package password

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/argon2"
)

const (
	TEST_PASSWORD       = "correct horse battery"
	TEST_OTHER_PASSWORD = "wrong horse battery"
	TEST_HASH_PREFIX    = "$argon2id$v=19$m=65536,t=3,p=2$"
)

// testSalt и testKey - валидные base64-части для ручной сборки хеша.
var (
	testSalt = base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef"))
	testKey  = base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
)

// buildHash собирает строку хеша из отдельных секций.
func buildHash(t *testing.T, name, version, params, salt, key string) string {
	t.Helper()
	return fmt.Sprintf("$%s$%s$%s$%s$%s", name, version, params, salt, key)
}

func TestHash_VerifyRoundTrip(t *testing.T) {
	encoded, err := Hash(TEST_PASSWORD)
	require.NoError(t, err)

	require.NoError(t, Verify(encoded, TEST_PASSWORD))
}

func TestVerify_WrongPassword(t *testing.T) {
	encoded, err := Hash(TEST_PASSWORD)
	require.NoError(t, err)

	require.ErrorIs(t, Verify(encoded, TEST_OTHER_PASSWORD), domain.ErrInvalidCredentials)
}

func TestHash_UniqueSalt(t *testing.T) {
	first, err := Hash(TEST_PASSWORD)
	require.NoError(t, err)
	second, err := Hash(TEST_PASSWORD)
	require.NoError(t, err)

	assert.NotEqual(t, first, second, "hashes of the same password must differ")
	assert.NoError(t, Verify(first, TEST_PASSWORD))
	assert.NoError(t, Verify(second, TEST_PASSWORD))
}

func TestHash_Format(t *testing.T) {
	encoded, err := Hash(TEST_PASSWORD)
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(encoded, TEST_HASH_PREFIX), "unexpected prefix: %s", encoded)

	parts := strings.Split(encoded, "$")
	require.Len(t, parts, domain.ARGON2_HASH_SECTIONS)
	assert.Empty(t, parts[0])

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	require.NoError(t, err)
	assert.Len(t, salt, domain.ARGON2_SALT_LENGTH)

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	require.NoError(t, err)
	assert.Len(t, key, domain.ARGON2_KEY_LENGTH)
}

func TestVerify_MalformedHash(t *testing.T) {
	err := Verify("garbage", "x")

	require.ErrorIs(t, err, domain.ErrInvalidPasswordHash)
	assert.NotErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestVerify_DegenerateHashNoPanic(t *testing.T) {
	encoded := buildHash(t, domain.ARGON2_ID_NAME, "v=19", "m=64,t=0,p=1", testSalt, testKey)

	var err error
	require.NotPanics(t, func() { err = Verify(encoded, TEST_PASSWORD) })
	require.ErrorIs(t, err, domain.ErrInvalidPasswordHash)
}

func TestDecode_WrongSectionCount(t *testing.T) {
	t.Parallel()

	_, err := decode("$argon2id$v=19$m=64,t=1,p=1$" + testSalt)

	require.ErrorIs(t, err, domain.ErrInvalidPasswordHash)
}

func TestDecode_NotArgon2id(t *testing.T) {
	t.Parallel()

	_, err := decode(buildHash(t, "argon2i", "v=19", "m=64,t=1,p=1", testSalt, testKey))

	require.ErrorIs(t, err, domain.ErrInvalidPasswordHash)
}

func TestDecode_BadVersionFormat(t *testing.T) {
	t.Parallel()

	_, err := decode(buildHash(t, domain.ARGON2_ID_NAME, "ver=19", "m=64,t=1,p=1", testSalt, testKey))

	require.ErrorIs(t, err, domain.ErrInvalidPasswordHash)
}

func TestDecode_UnsupportedVersion(t *testing.T) {
	t.Parallel()

	_, err := decode(buildHash(t, domain.ARGON2_ID_NAME, "v=16", "m=64,t=1,p=1", testSalt, testKey))

	require.ErrorIs(t, err, domain.ErrUnsupportedHashVersion)
}

func TestDecode_BadParams(t *testing.T) {
	t.Parallel()

	_, err := decode(buildHash(t, domain.ARGON2_ID_NAME, "v=19", "m=x,t=3,p=2", testSalt, testKey))

	require.ErrorIs(t, err, domain.ErrInvalidPasswordHash)
}

func TestDecode_ZeroTime(t *testing.T) {
	t.Parallel()

	_, err := decode(buildHash(t, domain.ARGON2_ID_NAME, "v=19", "m=64,t=0,p=1", testSalt, testKey))

	require.ErrorIs(t, err, domain.ErrInvalidPasswordHash)
}

func TestDecode_ZeroThreads(t *testing.T) {
	t.Parallel()

	_, err := decode(buildHash(t, domain.ARGON2_ID_NAME, "v=19", "m=64,t=1,p=0", testSalt, testKey))

	require.ErrorIs(t, err, domain.ErrInvalidPasswordHash)
}

func TestDecode_BadSaltBase64(t *testing.T) {
	t.Parallel()

	_, err := decode(buildHash(t, domain.ARGON2_ID_NAME, "v=19", "m=64,t=1,p=1", "!!!", testKey))

	require.ErrorIs(t, err, domain.ErrInvalidPasswordHash)
}

func TestDecode_BadKeyBase64(t *testing.T) {
	t.Parallel()

	_, err := decode(buildHash(t, domain.ARGON2_ID_NAME, "v=19", "m=64,t=1,p=1", testSalt, "!!!"))

	require.ErrorIs(t, err, domain.ErrInvalidPasswordHash)
}

func TestDecode_EmptySalt(t *testing.T) {
	t.Parallel()

	_, err := decode(buildHash(t, domain.ARGON2_ID_NAME, "v=19", "m=64,t=1,p=1", "", testKey))

	require.ErrorIs(t, err, domain.ErrInvalidPasswordHash)
}

func TestDecode_EmptyKey(t *testing.T) {
	t.Parallel()

	_, err := decode(buildHash(t, domain.ARGON2_ID_NAME, "v=19", "m=64,t=1,p=1", testSalt, ""))

	require.ErrorIs(t, err, domain.ErrInvalidPasswordHash)
}

func TestDecode_Valid(t *testing.T) {
	t.Parallel()

	want := params{
		time:    domain.ARGON2_TIME,
		memory:  domain.ARGON2_MEMORY,
		threads: domain.ARGON2_THREADS,
		salt:    []byte("0123456789abcdef"),
		key:     []byte("0123456789abcdef0123456789abcdef"),
	}

	got, err := decode(encode(want))
	require.NoError(t, err)

	assert.Equal(t, want.time, got.time)
	assert.Equal(t, want.memory, got.memory)
	assert.Equal(t, want.threads, got.threads)
	assert.Equal(t, want.salt, got.salt)
	assert.Equal(t, want.key, got.key)
}

func TestVerify_UsesStoredParams(t *testing.T) {
	t.Parallel()

	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte(TEST_PASSWORD), salt, 1, 64, 1, domain.ARGON2_KEY_LENGTH)
	encoded := encode(params{time: 1, memory: 64, threads: 1, salt: salt, key: key})

	require.NoError(t, Verify(encoded, TEST_PASSWORD))
}
