package domain

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidBirthDate = status.Error(codes.InvalidArgument, "birth_date must be in YYYY-MM-DD format")
)

var (
	ErrBearerTokenRequired  = errors.New("authorization header must be a bearer token")
	ErrAccessTokenNotValid  = errors.New("access token is not valid")
	ErrAuthenticationNeeded = errors.New("authentication is required")
	ErrRoleNotAllowed       = errors.New("role is not allowed to use this endpoint")
)

var (
	ErrJWKSURLRequired     = errors.New("jwks_url is empty")
	ErrSigningKeyNotFound  = errors.New("signing key is not published by the auth endpoint")
	ErrSigningKeyMalformed = errors.New("signing key from the auth endpoint is malformed")
)
