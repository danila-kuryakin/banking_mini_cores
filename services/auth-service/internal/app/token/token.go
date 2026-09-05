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

type Manager struct {
	privateKey *rsa.PrivateKey
	keyID      string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewManager(privateKeyPath string, accessTTL, refreshTTL time.Duration, logger *slog.Logger) (*Manager, error) {
	key, err := privateKey(privateKeyPath, logger)
	if err != nil {
		return nil, err
	}

	return &Manager{
		privateKey: key,
		keyID:      keyID(&key.PublicKey),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
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
	accessToken.Header[domain.JWK_KEY_ID_HEADER] = m.keyID

	access, err := accessToken.SignedString(m.privateKey)
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

	_, err := jwt.ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) {
		return &m.privateKey.PublicKey, nil
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
