package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID    uuid.UUID
	Role      models.Role
	ExpiresAt time.Time
}

type Pair struct {
	Access           string
	Refresh          string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type accessClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

// verifyKey - публичный ключ из набора вместе со своим kid.
type verifyKey struct {
	kid    string
	public *rsa.PublicKey
}

// Manager подписывает одним ключом, а проверяет любым из набора. Новый ключ
// становится активным, старый ещё какое-то время остаётся в наборе, пока
// не истекут подписанные им токены.
type Manager struct {
	signing    *rsa.PrivateKey
	signingKID string

	verify []verifyKey
	byKID  map[string]*rsa.PublicKey

	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewManager(keysDir string, accessTTL, refreshTTL time.Duration, logger *slog.Logger) (*Manager, error) {
	keys, err := loadKeys(keysDir, logger)
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, domain.ErrNoSigningKeys
	}

	m := &Manager{
		byKID:      make(map[string]*rsa.PublicKey, len(keys)),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}

	for _, k := range keys {
		kid := keyID(&k.key.PublicKey)

		m.verify = append(m.verify, verifyKey{kid: kid, public: &k.key.PublicKey})
		m.byKID[kid] = &k.key.PublicKey
	}

	active := keys[len(keys)-1]
	m.signing = active.key
	m.signingKID = keyID(&active.key.PublicKey)

	logger.Info("jwt signing keys loaded",
		"count", len(keys),
		"active_file", active.name,
		"active_kid", m.signingKID,
	)

	if len(keys) > domain.JWT_KEYS_EXPECTED {
		logger.Warn("more jwt keys than the rotation scheme expects, check for leftovers",
			"count", len(keys),
			"expected", domain.JWT_KEYS_EXPECTED,
		)
	}

	return m, nil
}

func (m *Manager) NewPair(user *models.User) (*Pair, error) {
	now := time.Now().UTC()
	accessExpiresAt := now.Add(m.accessTTL)

	claims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
		},
		Role: string(user.Role),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	accessToken.Header[domain.JWK_KEY_ID_HEADER] = m.signingKID

	access, err := accessToken.SignedString(m.signing)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refresh, err := newRefreshToken()
	if err != nil {
		return nil, err
	}

	return &Pair{
		Access:           access,
		Refresh:          refresh,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: now.Add(m.refreshTTL),
	}, nil
}

func (m *Manager) ParseAccess(raw string) (*Claims, error) {
	var claims accessClaims

	_, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header[domain.JWK_KEY_ID_HEADER].(string)

		public, ok := m.byKID[kid]
		if !ok {
			return nil, domain.ErrSigningKeyNotFound
		}

		return public, nil
	}, jwt.WithValidMethods([]string{domain.JWK_ALGORITHM}))
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	if claims.ExpiresAt == nil {
		return nil, domain.ErrInvalidToken
	}

	return &Claims{
		UserID:    userID,
		Role:      models.Role(claims.Role),
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func Hash(refreshToken string) []byte {
	sum := sha256.Sum256([]byte(refreshToken))

	return sum[:]
}

func newRefreshToken() (string, error) {
	buf := make([]byte, domain.REFRESH_TOKEN_BYTES)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}
