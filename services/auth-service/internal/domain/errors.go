package domain

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// repository - service
var (
	ErrEmailTaken           = errors.New("email already taken")
	ErrUserNotFound         = errors.New("user not found")
	ErrProfileAlreadyExists = errors.New("customer profile already exists")
	// ErrProfileNotCreated - маркер для хендлера: регистрация сорвалась именно на
	// заведении карточки клиента, а не на почте или пароле.
	ErrProfileNotCreated    = errors.New("customer profile was not created")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

var (
	ErrEmailRequired       = errors.New("email is required")
	ErrPasswordRequired    = errors.New("password is required")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidToken        = errors.New("invalid token")
	ErrLoginLocked         = errors.New("login is temporarily locked")
	ErrAdminEmailNotAdmin  = errors.New("default admin email belongs to a non-admin user")
)

var (
	ErrInvalidPasswordHash    = errors.New("password hash is malformed")
	ErrUnsupportedHashVersion = errors.New("unsupported password hash version")
	ErrPasswordHasherBusy     = errors.New("password hashing capacity exhausted")
)

var (
	ErrPrivateKeyNotPEM   = errors.New("jwt private key is not a valid pem block")
	ErrPrivateKeyNotRSA   = errors.New("jwt private key is not an rsa key")
	ErrNoSigningKeys      = errors.New("no jwt signing keys found")
	ErrSigningKeyNotFound = errors.New("signing key for the token kid is not in the set")
)

var (
	ErrIDMustBeUUID          = status.Error(codes.InvalidArgument, "id must be a uuid")
	ErrEmailAlreadyExists    = status.Error(codes.AlreadyExists, "email already registered")
	ErrProfileCreationFailed = status.Error(codes.Internal, "failed to create customer profile")
	ErrRefreshFailed         = status.Error(codes.Internal, "failed to refresh tokens")
	ErrLogoutFailed          = status.Error(codes.Internal, "failed to logout")
	ErrValidateFailed        = status.Error(codes.Internal, "failed to validate token")
	ErrDeleteUserFailed      = status.Error(codes.Internal, "failed to delete user")
	ErrUserDoesNotExist      = status.Error(codes.NotFound, "user not found")
	ErrTooManyLoginAttempts  = status.Error(codes.ResourceExhausted, "too many failed login attempts, try again later")
	ErrServerBusy            = status.Error(codes.ResourceExhausted, "server is busy, try again later")
)

var (
	ErrEmailIsRequired      = status.Error(codes.InvalidArgument, "email is required")
	ErrEmailIsNotValid      = status.Error(codes.InvalidArgument, "email is not a valid address")
	ErrEmailIsTooLong       = status.Error(codes.InvalidArgument, "email is longer than 254 characters")
	ErrPasswordIsRequired   = status.Error(codes.InvalidArgument, "password is required")
	ErrPasswordIsTooShort   = status.Error(codes.InvalidArgument, "password is shorter than 8 characters")
	ErrPasswordIsTooLong    = status.Error(codes.InvalidArgument, "password is longer than 72 characters")
	ErrRefreshTokenRequired = status.Error(codes.InvalidArgument, "refresh_token is required")
	ErrRefreshTokenTooLong  = status.Error(codes.InvalidArgument, "refresh_token is longer than 512 characters")
	ErrAccessTokenRequired  = status.Error(codes.InvalidArgument, "access_token is required")
	ErrAccessTokenTooLong   = status.Error(codes.InvalidArgument, "access_token is longer than 4096 characters")
)
