package domain

import (
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
)

const (
	ROLE_CLIENT  models.Role = "client"
	ROLE_OFFICER models.Role = "officer"
	ROLE_ADMIN   models.Role = "admin"
)

const (
	DEFAULT_ADMIN_EMAIL = "admin@banking.local"
)

const (
	DEFAULT_ACCESS_TTL          = 15 * time.Minute
	DEFAULT_REFRESH_TTL         = 720 * time.Hour
	DEFAULT_REQUEST_TIMEOUT     = 30 * time.Second
	DEFAULT_SHUTDOWN_TIMEOUT    = 5 * time.Second
	DEFAULT_READ_HEADER_TIMEOUT = 5 * time.Second
)

const (
	ARGON2_TIME        = 3
	ARGON2_MEMORY      = 64 * 1024
	ARGON2_THREADS     = 2
	ARGON2_KEY_LENGTH  = 32
	ARGON2_SALT_LENGTH = 16

	ARGON2_ID_NAME       = "argon2id"
	ARGON2_HASH_FORMAT   = "$%s$v=%d$m=%d,t=%d,p=%d$%s$%s"
	ARGON2_HASH_SECTIONS = 6
)

const (
	REFRESH_TOKEN_BYTES = 32

	RSA_KEY_BITS      = 2048
	JWK_KEY_TYPE      = "RSA"
	JWK_KEY_USE       = "sig"
	JWK_ALGORITHM     = "RS256"
	JWK_KEY_ID_HEADER = "kid"

	// PEM_RSA_PRIVATE_KEY - заголовок блока для ключа в формате PKCS#1.
	PEM_RSA_PRIVATE_KEY = "RSA PRIVATE KEY"
)

const (
	EMAIL_MAX_LENGTH   = 254
	EMAIL_PATTERN      = `^[^@\s]+@[^@\s]+\.[^@\s]+$`
	PASSWORD_MIN_RUNES = 8
	PASSWORD_MAX_RUNES = 72

	REFRESH_TOKEN_MAX_LENGTH = 512
	ACCESS_TOKEN_MAX_LENGTH  = 4096
)

const (
	JWKS_PATH           = "/.well-known/jwks.json"
	JWKS_CONTENT_TYPE   = "application/json"
	CONTENT_TYPE_HEADER = "Content-Type"
)
