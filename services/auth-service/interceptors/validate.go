package interceptors

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/auth/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

func Validate() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := validate(req); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func validate(req any) error {
	switch in := req.(type) {
	case *authv1.RegisterRequest:
		return validateCredentials(in.Email, in.Password)
	case *authv1.LoginRequest:
		return validateCredentials(in.Email, in.Password)
	case *authv1.RefreshRequest:
		return validateRefreshToken(in.RefreshToken)
	case *authv1.LogoutRequest:
		return validateRefreshToken(in.RefreshToken)
	case *authv1.ValidateTokenRequest:
		return validateAccessToken(in.AccessToken)
	case *authv1.DeleteUserRequest:
		return validateUserID(in.Id)
	default:
		return nil
	}
}

func validateCredentials(email, password string) error {
	if err := validateEmail(email); err != nil {
		return err
	}

	return validatePassword(password)
}

func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	var emailPattern = regexp.MustCompile(domain.EMAIL_PATTERN)

	switch {
	case email == "":
		return domain.ErrEmailIsRequired
	case len(email) > domain.EMAIL_MAX_LENGTH:
		return domain.ErrEmailIsTooLong
	case !emailPattern.MatchString(email):
		return domain.ErrEmailIsNotValid
	default:
		return nil
	}
}

func validatePassword(password string) error {
	length := utf8.RuneCountInString(password)

	switch {
	case password == "":
		return domain.ErrPasswordIsRequired
	case length < domain.PASSWORD_MIN_RUNES:
		return domain.ErrPasswordIsTooShort
	case length > domain.PASSWORD_MAX_RUNES:
		return domain.ErrPasswordIsTooLong
	default:
		return nil
	}
}

func validateRefreshToken(refreshToken string) error {
	switch {
	case refreshToken == "":
		return domain.ErrRefreshTokenRequired
	case len(refreshToken) > domain.REFRESH_TOKEN_MAX_LENGTH:
		return domain.ErrRefreshTokenTooLong
	default:
		return nil
	}
}

func validateAccessToken(accessToken string) error {
	switch {
	case accessToken == "":
		return domain.ErrAccessTokenRequired
	case len(accessToken) > domain.ACCESS_TOKEN_MAX_LENGTH:
		return domain.ErrAccessTokenTooLong
	default:
		return nil
	}
}

func validateUserID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return domain.ErrIDMustBeUUID
	}

	return nil
}
