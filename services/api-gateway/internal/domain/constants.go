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
	FILE_ID_PARAM = "file_id"
	SELF_ALIAS    = "me"
)

// Значения enum'ов filestore.v1 наружу отдаются доменными именами, без
// префикса: "passport" вместо "FILE_TYPE_PASSPORT".
const (
	FILE_TYPE_ENUM_PREFIX   = "FILE_TYPE_"
	FILE_STATUS_ENUM_PREFIX = "FILE_STATUS_"
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

// Счета и леджер: enum-ы наружу отдаются доменными именами без префикса.
const (
	ACCOUNT_ID_PARAM               = "account_id"
	TRANSACTION_ID_PARAM           = "transaction_id"
	ACCOUNT_STATUS_ENUM_PREFIX     = "ACCOUNT_STATUS_"
	TRANSACTION_TYPE_ENUM_PREFIX   = "TRANSACTION_TYPE_"
	TRANSACTION_STATUS_ENUM_PREFIX = "TRANSACTION_STATUS_"
	ENTRY_DIRECTION_ENUM_PREFIX    = "ENTRY_DIRECTION_"
)

const (
	APPLICATION_ID_PARAM           = "application_id"
	APPLICATION_STATUS_ENUM_PREFIX = "APPLICATION_STATUS_"
	DOCUMENT_TYPE_ENUM_PREFIX      = "DOCUMENT_TYPE_"
	CHECK_TYPE_ENUM_PREFIX         = "CHECK_TYPE_"
	CHECK_RESULT_ENUM_PREFIX       = "CHECK_RESULT_"
)
