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

// JWKS отдаёт весь набор: и активный ключ, и выводимые. Выводимый обязан
// оставаться опубликованным, пока не истёк последний подписанный им токен -
// иначе потребитель не сможет проверить подпись и вернёт 401.
func (m *Manager) JWKS() Set {
	keys := make([]JWK, 0, len(m.verify))

	for _, v := range m.verify {
		keys = append(keys, JWK{
			KeyType:   domain.JWK_KEY_TYPE,
			Use:       domain.JWK_KEY_USE,
			Algorithm: domain.JWK_ALGORITHM,
			KeyID:     v.kid,
			Modulus:   base64.RawURLEncoding.EncodeToString(v.public.N.Bytes()),
			Exponent:  base64.RawURLEncoding.EncodeToString(big.NewInt(int64(v.public.E)).Bytes()),
		})
	}

	return Set{Keys: keys}
}

// keyID считается от самого ключа, поэтому имена файлов и kid не связаны: один
// и тот же ключ всегда даёт один kid, а два разных не столкнутся.
func keyID(public *rsa.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(der)

	return base64.RawURLEncoding.EncodeToString(sum[:])
}
