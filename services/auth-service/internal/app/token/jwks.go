package token

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"math/big"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
)

type JWK struct {
	KeyType   string `json:"kty"`
	Use       string `json:"use"`
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Modulus   string `json:"n"`
	Exponent  string `json:"e"`
}

type Set struct {
	Keys []JWK `json:"keys"`
}

func (m *Manager) JWKS() Set {
	public := &m.privateKey.PublicKey

	return Set{
		Keys: []JWK{
			{
				KeyType:   domain.JWK_KEY_TYPE,
				Use:       domain.JWK_KEY_USE,
				Algorithm: domain.JWK_ALGORITHM,
				KeyID:     m.keyID,
				Modulus:   base64.RawURLEncoding.EncodeToString(public.N.Bytes()),
				Exponent:  base64.RawURLEncoding.EncodeToString(big.NewInt(int64(public.E)).Bytes()),
			},
		},
	}
}

func keyID(public *rsa.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(der)

	return base64.RawURLEncoding.EncodeToString(sum[:])
}
