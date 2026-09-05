package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID    string
	Role      string
	ExpiresAt time.Time
}

type accessClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

type jwk struct {
	KeyType   string `json:"kty"`
	KeyID     string `json:"kid"`
	Algorithm string `json:"alg"`
	Modulus   string `json:"n"`
	Exponent  string `json:"e"`
}

type set struct {
	Keys []jwk `json:"keys"`
}

type Verifier struct {
	url             string
	refreshInterval time.Duration
	client          *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

func NewVerifier(url string, refreshInterval time.Duration) *Verifier {
	return &Verifier{
		url:             url,
		refreshInterval: refreshInterval,
		client:          &http.Client{Timeout: domain.DEFAULT_JWKS_FETCH_TIMEOUT},
		keys:            make(map[string]*rsa.PublicKey),
	}
}

func (v *Verifier) Verify(ctx context.Context, accessToken string) (Claims, error) {
	var claims accessClaims

	_, err := jwt.ParseWithClaims(accessToken, &claims, v.keyFunc(ctx), jwt.WithValidMethods([]string{domain.JWT_ALGORITHM}))
	if err != nil {
		return Claims{}, domain.ErrAccessTokenNotValid
	}

	if claims.Subject == "" || claims.ExpiresAt == nil {
		return Claims{}, domain.ErrAccessTokenNotValid
	}

	return Claims{
		UserID:    claims.Subject,
		Role:      claims.Role,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func (v *Verifier) keyFunc(ctx context.Context) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		keyID, _ := token.Header[domain.JWT_KEY_ID_HEADER].(string)

		return v.publicKey(ctx, keyID)
	}
}

func (v *Verifier) publicKey(ctx context.Context, keyID string) (*rsa.PublicKey, error) {
	if key, ok := v.cached(keyID); ok {
		return key, nil
	}

	if err := v.refresh(ctx); err != nil {
		return nil, err
	}

	key, ok := v.cached(keyID)
	if !ok {
		return nil, domain.ErrSigningKeyNotFound
	}

	return key, nil
}

func (v *Verifier) cached(keyID string) (*rsa.PublicKey, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if time.Since(v.fetchedAt) > v.refreshInterval {
		return nil, false
	}

	key, ok := v.keys[keyID]

	return key, ok
}

func (v *Verifier) refresh(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.url, nil)
	if err != nil {
		return fmt.Errorf("build auth request: %w", err)
	}

	response, err := v.client.Do(request)
	if err != nil {
		return fmt.Errorf("fetch auth from %s: %w", v.url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch auth from %s: unexpected status %d", v.url, response.StatusCode)
	}

	var published set

	if err := json.NewDecoder(response.Body).Decode(&published); err != nil {
		return fmt.Errorf("decode auth from %s: %w", v.url, err)
	}

	keys, err := publicKeys(published)
	if err != nil {
		return err
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	v.keys = keys
	v.fetchedAt = time.Now()

	return nil
}

func publicKeys(published set) (map[string]*rsa.PublicKey, error) {
	keys := make(map[string]*rsa.PublicKey, len(published.Keys))

	for _, key := range published.Keys {
		if key.KeyType != domain.JWK_KEY_TYPE_RSA {
			continue
		}

		public, err := publicKey(key)
		if err != nil {
			return nil, err
		}

		keys[key.KeyID] = public
	}

	return keys, nil
}

func publicKey(key jwk) (*rsa.PublicKey, error) {
	modulus, err := base64.RawURLEncoding.DecodeString(key.Modulus)
	if err != nil {
		return nil, domain.ErrSigningKeyMalformed
	}

	exponent, err := base64.RawURLEncoding.DecodeString(key.Exponent)
	if err != nil {
		return nil, domain.ErrSigningKeyMalformed
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(modulus),
		E: int(new(big.Int).SetBytes(exponent).Int64()),
	}, nil
}
