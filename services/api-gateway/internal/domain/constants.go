package domain

import "time"

const (
	ROLE_CLIENT  = "client"
	ROLE_OFFICER = "officer"
	ROLE_ADMIN   = "admin"
)

const (
	AUTHORIZATION_HEADER   = "Authorization"
	AUTHORIZATION_METADATA = "authorization"
	BEARER_PREFIX          = "Bearer "
	ACTOR_KEY              = "actor"
)

const (
	JWT_ALGORITHM     = "RS256"
	JWT_KEY_ID_HEADER = "kid"
	JWK_KEY_TYPE_RSA  = "RSA"
)

const (
	BIRTH_DATE_LAYOUT             = "2006-01-02"
	DEFAULT_REQUEST_TIMEOUT       = 30 * time.Second
	DEFAULT_JWKS_REFRESH_INTERVAL = 5 * time.Minute
	DEFAULT_JWKS_FETCH_TIMEOUT    = 5 * time.Second
)
