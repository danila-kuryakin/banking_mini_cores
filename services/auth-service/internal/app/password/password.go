package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"golang.org/x/crypto/argon2"
)

type params struct {
	time    uint32
	memory  uint32
	threads uint8
	salt    []byte
	key     []byte
}

func Hash(plain string) (string, error) {
	salt := make([]byte, domain.ARGON2_SALT_LENGTH)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	key := argon2.IDKey(
		[]byte(plain),
		salt,
		domain.ARGON2_TIME,
		domain.ARGON2_MEMORY,
		domain.ARGON2_THREADS,
		domain.ARGON2_KEY_LENGTH,
	)

	return encode(params{
		time:    domain.ARGON2_TIME,
		memory:  domain.ARGON2_MEMORY,
		threads: domain.ARGON2_THREADS,
		salt:    salt,
		key:     key,
	}), nil
}

func Verify(encoded, plain string) error {
	stored, err := decode(encoded)
	if err != nil {
		return err
	}

	key := argon2.IDKey(
		[]byte(plain),
		stored.salt,
		stored.time,
		stored.memory,
		stored.threads,
		uint32(len(stored.key)),
	)

	if subtle.ConstantTimeCompare(key, stored.key) != 1 {
		return domain.ErrInvalidCredentials
	}

	return nil
}

func encode(p params) string {
	return fmt.Sprintf(
		domain.ARGON2_HASH_FORMAT,
		domain.ARGON2_ID_NAME,
		argon2.Version,
		p.memory,
		p.time,
		p.threads,
		base64.RawStdEncoding.EncodeToString(p.salt),
		base64.RawStdEncoding.EncodeToString(p.key),
	)
}

func decode(encoded string) (params, error) {
	sections := strings.Split(encoded, "$")
	if len(sections) != domain.ARGON2_HASH_SECTIONS || sections[1] != domain.ARGON2_ID_NAME {
		return params{}, domain.ErrInvalidPasswordHash
	}

	var version int
	if _, err := fmt.Sscanf(sections[2], "v=%d", &version); err != nil {
		return params{}, domain.ErrInvalidPasswordHash
	}

	if version != argon2.Version {
		return params{}, domain.ErrUnsupportedHashVersion
	}

	var p params
	if _, err := fmt.Sscanf(sections[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return params{}, domain.ErrInvalidPasswordHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(sections[4])
	if err != nil {
		return params{}, domain.ErrInvalidPasswordHash
	}

	key, err := base64.RawStdEncoding.DecodeString(sections[5])
	if err != nil {
		return params{}, domain.ErrInvalidPasswordHash
	}

	p.salt = salt
	p.key = key

	return p, nil
}
