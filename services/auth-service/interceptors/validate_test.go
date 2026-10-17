package interceptors

import (
	"context"
	"strings"
	"testing"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/auth/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	TEST_EMAIL                  = "user@example.com"
	TEST_EMAIL_DOMAIN           = "@example.com"
	TEST_CYRILLIC_PASSWORD_RUNE = "ж"
)

// handlerSpy запоминает вызов и отдаёт заранее заданный ответ.
type handlerSpy struct {
	called bool
	resp   any
}

func (h *handlerSpy) handle(_ context.Context, _ any) (any, error) {
	h.called = true
	return h.resp, nil
}

// validPassword отдаёт пароль минимальной допустимой длины.
func validPassword(t *testing.T) string {
	t.Helper()
	return strings.Repeat("p", domain.PASSWORD_MIN_RUNES)
}

// emailOfLength собирает email ровно заданной длины в байтах.
func emailOfLength(t *testing.T, length int) string {
	t.Helper()
	return strings.Repeat("a", length-len(TEST_EMAIL_DOMAIN)) + TEST_EMAIL_DOMAIN
}

// call прогоняет запрос через интерцептор со шпионом.
func call(t *testing.T, req any) (any, error, *handlerSpy) {
	t.Helper()
	spy := &handlerSpy{resp: &authv1.LoginResponse{}}
	resp, err := Validate()(context.Background(), req, &grpc.UnaryServerInfo{FullMethod: "/test"}, spy.handle)
	return resp, err, spy
}

// requireInvalid проверяет, что запрос отвергнут ожидаемой ошибкой без вызова handler.
func requireInvalid(t *testing.T, req any, want error) {
	t.Helper()
	resp, err, spy := call(t, req)
	require.ErrorIs(t, err, want)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Nil(t, resp)
	assert.False(t, spy.called, "handler must not be called on invalid request")
}

// requirePasses проверяет, что запрос дошёл до handler.
func requirePasses(t *testing.T, req any) {
	t.Helper()
	resp, err, spy := call(t, req)
	require.NoError(t, err)
	assert.True(t, spy.called, "handler must be called on valid request")
	assert.Same(t, spy.resp, resp)
}

func TestValidate_ValidRequestCallsHandler(t *testing.T) {
	t.Parallel()

	requirePasses(t, &authv1.RegisterRequest{Email: TEST_EMAIL, Password: validPassword(t)})
}

func TestValidate_InvalidRequestSkipsHandler(t *testing.T) {
	t.Parallel()

	resp, err, spy := call(t, &authv1.RegisterRequest{Email: "", Password: validPassword(t)})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.False(t, spy.called)
}

func TestValidate_UnknownRequestPasses(t *testing.T) {
	t.Parallel()

	requirePasses(t, &authv1.LogoutResponse{})
}

func TestValidate_EmailEmpty(t *testing.T) {
	t.Parallel()

	requireInvalid(t, &authv1.RegisterRequest{Email: "", Password: validPassword(t)}, domain.ErrEmailIsRequired)
}

func TestValidate_EmailOnlySpaces(t *testing.T) {
	t.Parallel()

	requireInvalid(t, &authv1.RegisterRequest{Email: "   ", Password: validPassword(t)}, domain.ErrEmailIsRequired)
}

func TestValidate_EmailMaxLength(t *testing.T) {
	t.Parallel()

	requirePasses(t, &authv1.RegisterRequest{Email: emailOfLength(t, domain.EMAIL_MAX_LENGTH), Password: validPassword(t)})
}

func TestValidate_EmailTooLong(t *testing.T) {
	t.Parallel()

	req := &authv1.RegisterRequest{Email: emailOfLength(t, domain.EMAIL_MAX_LENGTH+1), Password: validPassword(t)}
	requireInvalid(t, req, domain.ErrEmailIsTooLong)
}

func TestValidate_EmailNotValid(t *testing.T) {
	t.Parallel()

	requireInvalid(t, &authv1.RegisterRequest{Email: "userexample.com", Password: validPassword(t)}, domain.ErrEmailIsNotValid)
	requireInvalid(t, &authv1.RegisterRequest{Email: "user@example", Password: validPassword(t)}, domain.ErrEmailIsNotValid)
	requireInvalid(t, &authv1.RegisterRequest{Email: "us er@example.com", Password: validPassword(t)}, domain.ErrEmailIsNotValid)
}

func TestValidate_EmailTrimmed(t *testing.T) {
	t.Parallel()

	requirePasses(t, &authv1.RegisterRequest{Email: "  " + TEST_EMAIL + "  ", Password: validPassword(t)})
}

func TestValidate_PasswordEmpty(t *testing.T) {
	t.Parallel()

	requireInvalid(t, &authv1.RegisterRequest{Email: TEST_EMAIL, Password: ""}, domain.ErrPasswordIsRequired)
}

func TestValidate_PasswordTooShort(t *testing.T) {
	t.Parallel()

	req := &authv1.RegisterRequest{Email: TEST_EMAIL, Password: strings.Repeat("p", domain.PASSWORD_MIN_RUNES-1)}
	requireInvalid(t, req, domain.ErrPasswordIsTooShort)
}

func TestValidate_PasswordMinLength(t *testing.T) {
	t.Parallel()

	requirePasses(t, &authv1.RegisterRequest{Email: TEST_EMAIL, Password: strings.Repeat("p", domain.PASSWORD_MIN_RUNES)})
}

func TestValidate_PasswordMaxLength(t *testing.T) {
	t.Parallel()

	requirePasses(t, &authv1.RegisterRequest{Email: TEST_EMAIL, Password: strings.Repeat("p", domain.PASSWORD_MAX_RUNES)})
}

func TestValidate_PasswordTooLong(t *testing.T) {
	t.Parallel()

	req := &authv1.RegisterRequest{Email: TEST_EMAIL, Password: strings.Repeat("p", domain.PASSWORD_MAX_RUNES+1)}
	requireInvalid(t, req, domain.ErrPasswordIsTooLong)
}

func TestValidate_PasswordCountsRunes(t *testing.T) {
	t.Parallel()

	short := strings.Repeat(TEST_CYRILLIC_PASSWORD_RUNE, domain.PASSWORD_MIN_RUNES-1)
	require.Greater(t, len(short), domain.PASSWORD_MIN_RUNES, "byte length must exceed min runes")

	requirePasses(t, &authv1.RegisterRequest{Email: TEST_EMAIL, Password: strings.Repeat(TEST_CYRILLIC_PASSWORD_RUNE, domain.PASSWORD_MIN_RUNES)})
	requireInvalid(t, &authv1.RegisterRequest{Email: TEST_EMAIL, Password: short}, domain.ErrPasswordIsTooShort)
}

func TestValidate_EmailCheckedFirst(t *testing.T) {
	t.Parallel()

	requireInvalid(t, &authv1.RegisterRequest{Email: "userexample.com", Password: ""}, domain.ErrEmailIsNotValid)
}

func TestValidate_LoginUsesSameRules(t *testing.T) {
	t.Parallel()

	requireInvalid(t, &authv1.LoginRequest{Email: "", Password: validPassword(t)}, domain.ErrEmailIsRequired)
	short := strings.Repeat("p", domain.PASSWORD_MIN_RUNES-1)
	requireInvalid(t, &authv1.LoginRequest{Email: TEST_EMAIL, Password: short}, domain.ErrPasswordIsTooShort)
}

func TestValidate_RefreshTokenEmpty(t *testing.T) {
	t.Parallel()

	requireInvalid(t, &authv1.RefreshRequest{RefreshToken: ""}, domain.ErrRefreshTokenRequired)
	requireInvalid(t, &authv1.LogoutRequest{RefreshToken: ""}, domain.ErrRefreshTokenRequired)
}

func TestValidate_RefreshTokenMaxLength(t *testing.T) {
	t.Parallel()

	refreshToken := strings.Repeat("r", domain.REFRESH_TOKEN_MAX_LENGTH)
	requirePasses(t, &authv1.RefreshRequest{RefreshToken: refreshToken})
	requirePasses(t, &authv1.LogoutRequest{RefreshToken: refreshToken})
}

func TestValidate_RefreshTokenTooLong(t *testing.T) {
	t.Parallel()

	refreshToken := strings.Repeat("r", domain.REFRESH_TOKEN_MAX_LENGTH+1)
	requireInvalid(t, &authv1.RefreshRequest{RefreshToken: refreshToken}, domain.ErrRefreshTokenTooLong)
	requireInvalid(t, &authv1.LogoutRequest{RefreshToken: refreshToken}, domain.ErrRefreshTokenTooLong)
}

func TestValidate_AccessTokenEmpty(t *testing.T) {
	t.Parallel()

	requireInvalid(t, &authv1.ValidateTokenRequest{AccessToken: ""}, domain.ErrAccessTokenRequired)
}

func TestValidate_AccessTokenMaxLength(t *testing.T) {
	t.Parallel()

	requirePasses(t, &authv1.ValidateTokenRequest{AccessToken: strings.Repeat("a", domain.ACCESS_TOKEN_MAX_LENGTH)})
}

func TestValidate_AccessTokenTooLong(t *testing.T) {
	t.Parallel()

	req := &authv1.ValidateTokenRequest{AccessToken: strings.Repeat("a", domain.ACCESS_TOKEN_MAX_LENGTH+1)}
	requireInvalid(t, req, domain.ErrAccessTokenTooLong)
}

func TestValidate_DeleteUserNotUUID(t *testing.T) {
	t.Parallel()

	requireInvalid(t, &authv1.DeleteUserRequest{Id: "not-a-uuid"}, domain.ErrIDMustBeUUID)
}

func TestValidate_DeleteUserValid(t *testing.T) {
	t.Parallel()

	requirePasses(t, &authv1.DeleteUserRequest{Id: uuid.NewString()})
}

func TestValidate_EmailTrimmedBeforeLength(t *testing.T) {
	t.Parallel()

	email := "  " + emailOfLength(t, domain.EMAIL_MAX_LENGTH) + "  "
	requirePasses(t, &authv1.RegisterRequest{Email: email, Password: validPassword(t)})
}

func TestValidate_PasswordNotTrimmed(t *testing.T) {
	t.Parallel()

	password := " " + strings.Repeat("p", domain.PASSWORD_MIN_RUNES-1)
	requirePasses(t, &authv1.RegisterRequest{Email: TEST_EMAIL, Password: password})
}

func TestValidate_PasswordOnlySpaces(t *testing.T) {
	t.Parallel()

	requirePasses(t, &authv1.RegisterRequest{Email: TEST_EMAIL, Password: strings.Repeat(" ", domain.PASSWORD_MIN_RUNES)})
}
