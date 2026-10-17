package token

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_REFRESH_TOKEN_LENGTH = 43
	TEST_KEY_ID_LENGTH        = 43
	TEST_RSA_EXPONENT         = "AQAB"
	TEST_HASH_INPUT           = "refresh-token-value"
)

func TestNewPair_RoundTrip(t *testing.T) {
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: testKey(t, 0)})
	user := testUser(t)

	pair, err := manager.NewPair(user)
	require.NoError(t, err)
	claims, err := manager.ParseAccess(pair.Access)
	require.NoError(t, err)

	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Role, claims.Role)
	assert.WithinDuration(t, pair.AccessExpiresAt, claims.ExpiresAt, time.Second)
}

func TestNewPair_TTL(t *testing.T) {
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: testKey(t, 0)})

	before := time.Now()
	pair, err := manager.NewPair(testUser(t))
	after := time.Now()
	require.NoError(t, err)

	assert.False(t, pair.AccessExpiresAt.Before(before.Add(TEST_ACCESS_TTL)), "access expiry too early")
	assert.False(t, pair.AccessExpiresAt.After(after.Add(TEST_ACCESS_TTL)), "access expiry too late")
	assert.False(t, pair.RefreshExpiresAt.Before(before.Add(TEST_REFRESH_TTL)), "refresh expiry too early")
	assert.False(t, pair.RefreshExpiresAt.After(after.Add(TEST_REFRESH_TTL)), "refresh expiry too late")
}

func TestNewPair_KidHeader(t *testing.T) {
	key := testKey(t, 0)
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: key})

	pair, err := manager.NewPair(testUser(t))
	require.NoError(t, err)

	kid := headerKid(t, pair.Access)
	assert.Equal(t, keyID(&key.PublicKey), kid)
	assert.Contains(t, jwksKids(t, manager.JWKS()), kid)
}

func TestNewPair_RefreshToken(t *testing.T) {
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: testKey(t, 0)})

	first, err := manager.NewPair(testUser(t))
	require.NoError(t, err)
	second, err := manager.NewPair(testUser(t))
	require.NoError(t, err)

	assert.Len(t, first.Refresh, TEST_REFRESH_TOKEN_LENGTH)
	raw, err := base64.RawURLEncoding.DecodeString(first.Refresh)
	require.NoError(t, err)
	assert.Len(t, raw, domain.REFRESH_TOKEN_BYTES)
	assert.NotEqual(t, first.Refresh, second.Refresh)
}

func TestParseAccess_Expired(t *testing.T) {
	manager := newTestManager(t, -time.Minute, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: testKey(t, 0)})

	pair, err := manager.NewPair(testUser(t))
	require.NoError(t, err)
	_, err = manager.ParseAccess(pair.Access)

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestParseAccess_UnknownKid(t *testing.T) {
	issuer := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: testKey(t, 0)})
	verifier := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: testKey(t, 1)})

	pair, err := issuer.NewPair(testUser(t))
	require.NoError(t, err)
	_, err = verifier.ParseAccess(pair.Access)

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestParseAccess_MissingKid(t *testing.T) {
	key := testKey(t, 0)
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: key})

	raw := signRS256(t, key, "", validClaims(t))
	_, err := manager.ParseAccess(raw)

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestParseAccess_HS256WithPublicKey(t *testing.T) {
	key := testKey(t, 0)
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: key})
	secret, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims(t))
	token.Header[domain.JWK_KEY_ID_HEADER] = keyID(&key.PublicKey)
	raw, err := token.SignedString(secret)
	require.NoError(t, err)
	_, err = manager.ParseAccess(raw)

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestParseAccess_AlgNone(t *testing.T) {
	key := testKey(t, 0)
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: key})

	token := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims(t))
	token.Header[domain.JWK_KEY_ID_HEADER] = keyID(&key.PublicKey)
	raw, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	_, err = manager.ParseAccess(raw)

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestParseAccess_TamperedPayload(t *testing.T) {
	key := testKey(t, 0)
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: key})
	pair, err := manager.NewPair(testUser(t))
	require.NoError(t, err)

	claims := validClaims(t)
	claims["role"] = string(domain.ROLE_ADMIN)
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	parts := strings.Split(pair.Access, ".")
	require.Len(t, parts, 3)
	parts[1] = base64.RawURLEncoding.EncodeToString(payload)

	_, err = manager.ParseAccess(strings.Join(parts, "."))

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestParseAccess_SubjectNotUUID(t *testing.T) {
	key := testKey(t, 0)
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: key})
	claims := validClaims(t)
	claims["sub"] = "not-a-uuid"

	raw := signRS256(t, key, keyID(&key.PublicKey), claims)
	_, err := manager.ParseAccess(raw)

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestParseAccess_MissingExp(t *testing.T) {
	key := testKey(t, 0)
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: key})
	claims := validClaims(t)
	delete(claims, "exp")

	raw := signRS256(t, key, keyID(&key.PublicKey), claims)
	_, err := manager.ParseAccess(raw)

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestParseAccess_Garbage(t *testing.T) {
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: testKey(t, 0)})

	_, err := manager.ParseAccess("abc")

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestParseAccess_OldKeyStillVerifies(t *testing.T) {
	keyA, keyB := testKey(t, 0), testKey(t, 1)
	issuer := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: keyA})
	verifier := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{
		TEST_KEY_FILE_A: keyA,
		TEST_KEY_FILE_B: keyB,
	})
	user := testUser(t)

	pair, err := issuer.NewPair(user)
	require.NoError(t, err)
	claims, err := verifier.ParseAccess(pair.Access)

	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
}

func TestParseAccess_RemovedKeyRejected(t *testing.T) {
	issuer := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: testKey(t, 0)})
	verifier := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_B: testKey(t, 1)})

	pair, err := issuer.NewPair(testUser(t))
	require.NoError(t, err)
	_, err = verifier.ParseAccess(pair.Access)

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestJWKS_Fields(t *testing.T) {
	key := testKey(t, 0)
	manager := newTestManager(t, TEST_ACCESS_TTL, map[string]*rsa.PrivateKey{TEST_KEY_FILE_A: key})

	set := manager.JWKS()
	require.Len(t, set.Keys, 1)
	jwk := set.Keys[0]

	assert.Equal(t, domain.JWK_KEY_TYPE, jwk.KeyType)
	assert.Equal(t, domain.JWK_KEY_USE, jwk.Use)
	assert.Equal(t, domain.JWK_ALGORITHM, jwk.Algorithm)
	assert.Equal(t, keyID(&key.PublicKey), jwk.KeyID)
	assert.Equal(t, TEST_RSA_EXPONENT, jwk.Exponent)

	modulus, err := base64.RawURLEncoding.DecodeString(jwk.Modulus)
	require.NoError(t, err)
	exponent, err := base64.RawURLEncoding.DecodeString(jwk.Exponent)
	require.NoError(t, err)
	public := &rsa.PublicKey{
		N: new(big.Int).SetBytes(modulus),
		E: int(new(big.Int).SetBytes(exponent).Int64()),
	}
	assert.True(t, public.Equal(&key.PublicKey), "public key from JWKS differs from source key")
}

func TestKeyID_Deterministic(t *testing.T) {
	keyA, keyB := testKey(t, 0), testKey(t, 1)

	first := keyID(&keyA.PublicKey)
	second := keyID(&keyA.PublicKey)

	assert.Equal(t, first, second)
	assert.Len(t, first, TEST_KEY_ID_LENGTH)
	assert.NotEqual(t, first, keyID(&keyB.PublicKey))
}

func TestHash_Sha256(t *testing.T) {
	want := sha256.Sum256([]byte(TEST_HASH_INPUT))

	got := Hash(TEST_HASH_INPUT)

	assert.Len(t, got, sha256.Size)
	assert.Equal(t, want[:], got)
	assert.Equal(t, got, Hash(TEST_HASH_INPUT))
}
