package server

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/password"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/mocks"
	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/auth/v1"
	commonv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/common/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	TEST_EMAIL         = "user@example.com"
	TEST_PASSWORD      = "correct horse battery"
	TEST_REFRESH_TOKEN = "test-refresh-token"
	TEST_TIMEOUT       = 5 * time.Second
	TEST_ACCESS_TTL    = 15 * time.Minute
	TEST_REFRESH_TTL   = 720 * time.Hour
)

// testTime - фиксированное время для моделей из моков.
var testTime = time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

// Хеш TEST_PASSWORD считается один раз на пакет: argon2 дорогой.
var (
	testHashOnce sync.Once
	testHash     string
	testHashErr  error
)

// passwordHash отдаёт закешированный хеш TEST_PASSWORD.
func passwordHash(t *testing.T) string {
	t.Helper()
	testHashOnce.Do(func() { testHash, testHashErr = password.Hash(TEST_PASSWORD) })
	require.NoError(t, testHashErr)
	return testHash
}

// testEnv - gRPC-сервер над настоящим сервисом с моками репозитория и customer-service.
type testEnv struct {
	server    *AuthServer
	repo      *mocks.AuthRepositoryMock
	customers *mocks.CustomerProfilesMock
	manager   *token.Manager
}

// newTestEnv собирает AuthServer над свежими моками.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	manager, err := token.NewManager("", TEST_ACCESS_TTL, TEST_REFRESH_TTL, logger)
	require.NoError(t, err)
	repo := &mocks.AuthRepositoryMock{}
	customers := &mocks.CustomerProfilesMock{}
	svc := service.NewService(&repository.Repository{Auth: repo}, manager, customers)
	return &testEnv{
		server:    NewAuthServer(svc, logger, TEST_TIMEOUT),
		repo:      repo,
		customers: customers,
		manager:   manager,
	}
}

// assertExpectations проверяет ожидания всех моков окружения.
func (e *testEnv) assertExpectations(t *testing.T) {
	t.Helper()
	e.repo.AssertExpectations(t)
	e.customers.AssertExpectations(t)
}

// testUser отдаёт пользователя с хешем TEST_PASSWORD и заданной ролью.
func testUser(t *testing.T, role models.Role) *models.User {
	t.Helper()
	return &models.User{
		ID:           uuid.New(),
		Email:        TEST_EMAIL,
		PasswordHash: passwordHash(t),
		Role:         role,
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
}

// activeRefreshToken отдаёт неотозванный токен, истекающий через час.
func activeRefreshToken(t *testing.T, userID uuid.UUID) *models.RefreshToken {
	t.Helper()
	return &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: token.Hash(TEST_REFRESH_TOKEN),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
		CreatedAt: testTime,
	}
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	created := testUser(t, domain.ROLE_CLIENT)
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(created, nil).Once()
	env.customers.On("CreateProfile", mock.Anything, created.ID).Return(nil).Once()

	resp, err := env.server.Register(context.Background(), &authv1.RegisterRequest{Email: TEST_EMAIL, Password: TEST_PASSWORD})

	require.NoError(t, err)
	assert.Equal(t, created.ID.String(), resp.GetId())
	assert.Equal(t, TEST_EMAIL, resp.GetEmail())
	assert.Equal(t, commonv1.Role_ROLE_CLIENT, resp.GetRole())
	require.NotNil(t, resp.GetCreatedAt())
	assert.True(t, testTime.Equal(resp.GetCreatedAt().AsTime()), "created_at mismatch")
}

func TestRegister_ProfileNotCreated(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	created := testUser(t, domain.ROLE_CLIENT)
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(created, nil).Once()
	env.customers.On("CreateProfile", mock.Anything, created.ID).Return(errors.New("customer down")).Once()
	env.repo.On("DeleteUser", mock.Anything, created.ID).Return(nil).Once()

	_, err := env.server.Register(context.Background(), &authv1.RegisterRequest{Email: TEST_EMAIL, Password: TEST_PASSWORD})

	require.ErrorIs(t, err, domain.ErrProfileCreationFailed)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestRegister_EmailTaken(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil, domain.ErrEmailTaken).Once()

	_, err := env.server.Register(context.Background(), &authv1.RegisterRequest{Email: TEST_EMAIL, Password: TEST_PASSWORD})

	require.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestRegister_EmptyEmail(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	_, err := env.server.Register(context.Background(), &authv1.RegisterRequest{Email: "", Password: TEST_PASSWORD})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Equal(t, "email is required", status.Convert(err).Message())
}

func TestRegister_RepoError(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil, errors.New("db down")).Once()

	_, err := env.server.Register(context.Background(), &authv1.RegisterRequest{Email: TEST_EMAIL, Password: TEST_PASSWORD})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Equal(t, "failed to register user", status.Convert(err).Message())
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	stored := testUser(t, domain.ROLE_CLIENT)
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(stored, nil).Once()
	env.repo.On("AddRefreshToken", mock.Anything, mock.Anything).Return(nil).Once()

	resp, err := env.server.Login(context.Background(), &authv1.LoginRequest{Email: TEST_EMAIL, Password: TEST_PASSWORD})

	require.NoError(t, err)
	require.NotNil(t, resp.GetUser())
	assert.Equal(t, stored.ID.String(), resp.GetUser().GetId())
	assert.Equal(t, TEST_EMAIL, resp.GetUser().GetEmail())
	assert.Equal(t, commonv1.Role_ROLE_CLIENT, resp.GetUser().GetRole())
	assert.NotNil(t, resp.GetUser().GetCreatedAt())
	tokens := resp.GetTokens()
	require.NotNil(t, tokens)
	claims, err := env.manager.ParseAccess(tokens.GetAccessToken())
	require.NoError(t, err)
	assert.Equal(t, stored.ID, claims.UserID)
	assert.NotEmpty(t, tokens.GetRefreshToken())
	assert.NotNil(t, tokens.GetAccessExpiresAt())
	assert.NotNil(t, tokens.GetRefreshExpiresAt())
}

func TestLogin_InvalidCredentials(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(nil, domain.ErrUserNotFound).Once()

	_, err := env.server.Login(context.Background(), &authv1.LoginRequest{Email: TEST_EMAIL, Password: TEST_PASSWORD})

	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestLogin_RepoError(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(nil, errors.New("db down")).Once()

	_, err := env.server.Login(context.Background(), &authv1.LoginRequest{Email: TEST_EMAIL, Password: TEST_PASSWORD})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Equal(t, "failed to login", status.Convert(err).Message())
}

func TestLogin_SetsDeadline(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	var gotCtx context.Context
	var calledAt time.Time
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).
		Run(func(args mock.Arguments) {
			gotCtx = args.Get(0).(context.Context)
			calledAt = time.Now()
		}).
		Return(nil, domain.ErrUserNotFound).Once()

	_, _ = env.server.Login(context.Background(), &authv1.LoginRequest{Email: TEST_EMAIL, Password: TEST_PASSWORD})

	require.NotNil(t, gotCtx)
	deadline, ok := gotCtx.Deadline()
	require.True(t, ok, "repository context must have a deadline")
	assert.False(t, deadline.After(calledAt.Add(TEST_TIMEOUT)), "deadline is later than call time plus timeout")
}

// --- Refresh ---

func TestRefresh_Success(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	user := testUser(t, domain.ROLE_CLIENT)
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(activeRefreshToken(t, user.ID), nil).Once()
	env.repo.On("GetUserByID", mock.Anything, user.ID).Return(user, nil).Once()
	env.repo.On("RotateRefreshToken", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	resp, err := env.server.Refresh(context.Background(), &authv1.RefreshRequest{RefreshToken: TEST_REFRESH_TOKEN})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.GetAccessToken())
	assert.NotEmpty(t, resp.GetRefreshToken())
	assert.NotNil(t, resp.GetAccessExpiresAt())
	assert.NotNil(t, resp.GetRefreshExpiresAt())
}

func TestRefresh_Invalid(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(nil, domain.ErrRefreshTokenNotFound).Once()

	_, err := env.server.Refresh(context.Background(), &authv1.RefreshRequest{RefreshToken: TEST_REFRESH_TOKEN})

	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.Equal(t, "invalid refresh token", status.Convert(err).Message())
}

func TestRefresh_RepoError(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(nil, errors.New("db down")).Once()

	_, err := env.server.Refresh(context.Background(), &authv1.RefreshRequest{RefreshToken: TEST_REFRESH_TOKEN})

	require.ErrorIs(t, err, domain.ErrRefreshFailed)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// --- Logout ---

func TestLogout_Success(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("RevokeRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(nil).Once()

	resp, err := env.server.Logout(context.Background(), &authv1.LogoutRequest{RefreshToken: TEST_REFRESH_TOKEN})

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestLogout_Invalid(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("RevokeRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(domain.ErrRefreshTokenNotFound).Once()

	_, err := env.server.Logout(context.Background(), &authv1.LogoutRequest{RefreshToken: TEST_REFRESH_TOKEN})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Equal(t, "invalid refresh token", status.Convert(err).Message())
}

func TestLogout_RepoError(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("RevokeRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(errors.New("db down")).Once()

	_, err := env.server.Logout(context.Background(), &authv1.LogoutRequest{RefreshToken: TEST_REFRESH_TOKEN})

	require.ErrorIs(t, err, domain.ErrLogoutFailed)
	assert.Equal(t, codes.Internal, status.Code(err))
}

// --- ValidateToken ---

func TestValidateToken_Valid(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	user := testUser(t, domain.ROLE_OFFICER)
	pair, err := env.manager.NewPair(user)
	require.NoError(t, err)

	resp, err := env.server.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{AccessToken: pair.Access})

	require.NoError(t, err)
	assert.True(t, resp.GetValid())
	assert.Equal(t, user.ID.String(), resp.GetId())
	assert.Equal(t, commonv1.Role_ROLE_OFFICER, resp.GetRole())
	require.NotNil(t, resp.GetExpiresAt())
	assert.WithinDuration(t, pair.AccessExpiresAt, resp.GetExpiresAt().AsTime(), time.Second)
}

func TestValidateToken_Invalid(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	resp, err := env.server.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{AccessToken: "garbage"})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.GetValid())
}

func TestValidateToken_Empty(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	resp, err := env.server.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{AccessToken: ""})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.GetValid())
}

// --- CreateOfficer ---

func TestCreateOfficer_Success(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(testUser(t, domain.ROLE_OFFICER), nil).Once()

	resp, err := env.server.CreateOfficer(context.Background(), &authv1.RegisterRequest{Email: TEST_EMAIL, Password: TEST_PASSWORD})

	require.NoError(t, err)
	assert.Equal(t, commonv1.Role_ROLE_OFFICER, resp.GetRole())
	env.customers.AssertNotCalled(t, "CreateProfile", mock.Anything, mock.Anything)
}

func TestCreateOfficer_WrongRole(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(testUser(t, domain.ROLE_CLIENT), nil).Once()

	_, err := env.server.CreateOfficer(context.Background(), &authv1.RegisterRequest{Email: TEST_EMAIL, Password: TEST_PASSWORD})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Equal(t, "invalid role", status.Convert(err).Message())
}

func TestCreateOfficer_EmailTaken(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil, domain.ErrEmailTaken).Once()

	_, err := env.server.CreateOfficer(context.Background(), &authv1.RegisterRequest{Email: TEST_EMAIL, Password: TEST_PASSWORD})

	require.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

// --- DeleteUser ---

func TestDeleteUser_Success(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	userID := uuid.New()
	env.repo.On("DeleteUser", mock.Anything, userID).Return(nil).Once()

	resp, err := env.server.DeleteUser(context.Background(), &authv1.DeleteUserRequest{Id: userID.String()})

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestDeleteUser_BadUUID(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	_, err := env.server.DeleteUser(context.Background(), &authv1.DeleteUserRequest{Id: "not-a-uuid"})

	require.ErrorIs(t, err, domain.ErrIDMustBeUUID)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	env.repo.AssertNotCalled(t, "DeleteUser", mock.Anything, mock.Anything)
}

func TestDeleteUser_RepoError(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	userID := uuid.New()
	env.repo.On("DeleteUser", mock.Anything, userID).Return(errors.New("db down")).Once()

	_, err := env.server.DeleteUser(context.Background(), &authv1.DeleteUserRequest{Id: userID.String()})

	require.ErrorIs(t, err, domain.ErrDeleteUserFailed)
	assert.Equal(t, codes.Internal, status.Code(err))
}
