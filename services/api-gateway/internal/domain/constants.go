package domain

import "time"

const (
	SWAGGER_ROUTE = "/swagger/*any"
	SWAGGER_PAGE  = "/swagger/index.html"
)

const (
	ROLE_CLIENT  = "client"
	ROLE_OFFICER = "officer"
	ROLE_ADMIN   = "admin"

	ROLE_ENUM_PREFIX = "ROLE_"
)

const (
	AUTHORIZATION_HEADER   = "Authorization"
	AUTHORIZATION_METADATA = "authorization"
	BEARER_PREFIX          = "Bearer "
	ACTOR_KEY              = "actor"

	USER_ID_PARAM = "user_id"
	SELF_ALIAS    = "me"
)

const (
	JWT_ALGORITHM     = "RS256"
	JWT_KEY_ID_HEADER = "kid"
	JWK_KEY_TYPE_RSA  = "RSA"
)

const (
	DEFAULT_REQUEST_TIMEOUT       = 30 * time.Second
	DEFAULT_SHUTDOWN_TIMEOUT      = 5 * time.Second
	DEFAULT_JWKS_REFRESH_INTERVAL = 5 * time.Minute
	DEFAULT_JWKS_FETCH_TIMEOUT    = 5 * time.Second
)
