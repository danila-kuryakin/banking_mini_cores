package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/password"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	TEST_EMAIL          = "user@example.com"
	TEST_RAW_EMAIL      = "  User@Example.COM "
	TEST_PASSWORD       = "correct horse battery"
	TEST_WRONG_PASSWORD = "wrong horse battery"
	TEST_REFRESH_TOKEN  = "test-refresh-token"
	TEST_ACCESS_TTL     = 15 * time.Minute
	TEST_REFRESH_TTL    = 720 * time.Hour
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

// testEnv - сервис с моками и настоящим менеджером токенов.
type testEnv struct {
	service   *AuthService
	repo      *mocks.AuthRepositoryMock
	customers *mocks.CustomerProfilesMock
	manager   *token.Manager
}

// newTestEnv собирает AuthService над свежими моками.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	manager, err := token.NewManager("", TEST_ACCESS_TTL, TEST_REFRESH_TTL, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	repo := &mocks.AuthRepositoryMock{}
	customers := &mocks.CustomerProfilesMock{}
	return &testEnv{
		service:   NewAuthService(&repository.Repository{Auth: repo}, manager, customers),
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

// storedRefreshToken отдаёт запись refresh-токена с заданным сроком и отзывом.
func storedRefreshToken(t *testing.T, userID uuid.UUID, expiresAt time.Time, revokedAt *time.Time) *models.RefreshToken {
	t.Helper()
	return &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: token.Hash(TEST_REFRESH_TOKEN),
		ExpiresAt: expiresAt,
		RevokedAt: revokedAt,
		CreatedAt: testTime,
	}
}

// activeRefreshToken отдаёт неотозванный токен, истекающий через час.
func activeRefreshToken(t *testing.T, userID uuid.UUID) *models.RefreshToken {
	t.Helper()
	return storedRefreshToken(t, userID, time.Now().UTC().Add(time.Hour), nil)
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	created := testUser(t, domain.ROLE_CLIENT)
	var gotUser models.User
	env.repo.On("CreateUser", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { gotUser = args.Get(1).(models.User) }).
		Return(created, nil).Once()
	env.customers.On("CreateProfile", mock.Anything, created.ID).Return(nil).Once()

	user, err := env.service.Register(context.Background(), TEST_RAW_EMAIL, TEST_PASSWORD)

	require.NoError(t, err)
	assert.Same(t, created, user)
	assert.Equal(t, TEST_EMAIL, gotUser.Email)
	assert.Equal(t, domain.ROLE_CLIENT, gotUser.Role)
	assert.NoError(t, password.Verify(gotUser.PasswordHash, TEST_PASSWORD))
	env.repo.AssertNotCalled(t, "DeleteUser", mock.Anything, mock.Anything)
}

func TestRegister_EmptyEmail(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	_, err := env.service.Register(context.Background(), "   ", TEST_PASSWORD)

	require.ErrorIs(t, err, domain.ErrEmailRequired)
	env.repo.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything)
}

func TestRegister_EmptyPassword(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	_, err := env.service.Register(context.Background(), TEST_EMAIL, "")

	require.ErrorIs(t, err, domain.ErrPasswordRequired)
	env.repo.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything)
}

func TestRegister_EmailTaken(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil, domain.ErrEmailTaken).Once()

	_, err := env.service.Register(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.ErrorIs(t, err, domain.ErrEmailTaken)
	env.customers.AssertNotCalled(t, "CreateProfile", mock.Anything, mock.Anything)
}

func TestRegister_ProfileAlreadyExists(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	created := testUser(t, domain.ROLE_CLIENT)
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(created, nil).Once()
	env.customers.On("CreateProfile", mock.Anything, created.ID).Return(domain.ErrProfileAlreadyExists).Once()

	user, err := env.service.Register(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.NoError(t, err)
	assert.Same(t, created, user)
	env.repo.AssertNotCalled(t, "DeleteUser", mock.Anything, mock.Anything)
}

func TestRegister_ProfileFailed(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	created := testUser(t, domain.ROLE_CLIENT)
	cause := errors.New("customer down")
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(created, nil).Once()
	env.customers.On("CreateProfile", mock.Anything, created.ID).Return(cause).Once()
	env.repo.On("DeleteUser", mock.Anything, created.ID).Return(nil).Once()

	user, err := env.service.Register(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.ErrorIs(t, err, domain.ErrProfileNotCreated)
	require.ErrorIs(t, err, cause)
	assert.Nil(t, user)
}

func TestRegister_ProfileFailedAndDeleteFailed(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	created := testUser(t, domain.ROLE_CLIENT)
	cause := errors.New("customer down")
	deleteErr := errors.New("delete failed")
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(created, nil).Once()
	env.customers.On("CreateProfile", mock.Anything, created.ID).Return(cause).Once()
	env.repo.On("DeleteUser", mock.Anything, created.ID).Return(deleteErr).Once()

	user, err := env.service.Register(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.ErrorIs(t, err, domain.ErrProfileNotCreated)
	require.ErrorIs(t, err, cause)
	require.ErrorIs(t, err, deleteErr)
	assert.Nil(t, user)
}

// --- CreateOfficer ---

func TestCreateOfficer_Success(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	created := testUser(t, domain.ROLE_OFFICER)
	var gotUser models.User
	env.repo.On("CreateUser", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { gotUser = args.Get(1).(models.User) }).
		Return(created, nil).Once()

	user, err := env.service.CreateOfficer(context.Background(), TEST_RAW_EMAIL, TEST_PASSWORD)

	require.NoError(t, err)
	assert.Same(t, created, user)
	assert.Equal(t, domain.ROLE_OFFICER, gotUser.Role)
	assert.Equal(t, TEST_EMAIL, gotUser.Email)
	env.customers.AssertNotCalled(t, "CreateProfile", mock.Anything, mock.Anything)
}

func TestCreateOfficer_EmailTaken(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil, domain.ErrEmailTaken).Once()

	_, err := env.service.CreateOfficer(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.ErrorIs(t, err, domain.ErrEmailTaken)
}

// --- EnsureAdmin ---

func TestEnsureAdmin_EmptyEmail(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	created, err := env.service.EnsureAdmin(context.Background(), "  ", TEST_PASSWORD)

	require.ErrorIs(t, err, domain.ErrEmailRequired)
	assert.False(t, created)
	env.repo.AssertNotCalled(t, "GetUserByEmail", mock.Anything, mock.Anything)
}

func TestEnsureAdmin_EmptyPassword(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	created, err := env.service.EnsureAdmin(context.Background(), TEST_EMAIL, "")

	require.ErrorIs(t, err, domain.ErrPasswordRequired)
	assert.False(t, created)
	env.repo.AssertNotCalled(t, "GetUserByEmail", mock.Anything, mock.Anything)
}

func TestEnsureAdmin_AlreadyExists(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(testUser(t, domain.ROLE_ADMIN), nil).Once()

	created, err := env.service.EnsureAdmin(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.NoError(t, err)
	assert.False(t, created)
	env.repo.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything)
}

func TestEnsureAdmin_Creates(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	var gotUser models.User
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(nil, domain.ErrUserNotFound).Once()
	env.repo.On("CreateUser", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { gotUser = args.Get(1).(models.User) }).
		Return(testUser(t, domain.ROLE_ADMIN), nil).Once()

	created, err := env.service.EnsureAdmin(context.Background(), TEST_RAW_EMAIL, TEST_PASSWORD)

	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, domain.ROLE_ADMIN, gotUser.Role)
	assert.Equal(t, TEST_EMAIL, gotUser.Email)
}

func TestEnsureAdmin_LookupError(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	lookupErr := errors.New("db down")
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(nil, lookupErr).Once()

	created, err := env.service.EnsureAdmin(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.ErrorIs(t, err, lookupErr)
	assert.False(t, created)
	env.repo.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything)
}

func TestEnsureAdmin_CreateFails(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	createErr := errors.New("insert failed")
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(nil, domain.ErrUserNotFound).Once()
	env.repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil, createErr).Once()

	created, err := env.service.EnsureAdmin(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.ErrorIs(t, err, createErr)
	assert.False(t, created)
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	stored := testUser(t, domain.ROLE_CLIENT)
	var gotToken models.RefreshToken
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(stored, nil).Once()
	env.repo.On("AddRefreshToken", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { gotToken = args.Get(1).(models.RefreshToken) }).
		Return(nil).Once()

	user, pair, err := env.service.Login(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.NoError(t, err)
	require.NotNil(t, pair)
	assert.Same(t, stored, user)
	claims, err := env.manager.ParseAccess(pair.Access)
	require.NoError(t, err)
	assert.Equal(t, stored.ID, claims.UserID)
	assert.Equal(t, stored.Role, claims.Role)
	assert.Equal(t, stored.ID, gotToken.UserID)
	assert.Equal(t, token.Hash(pair.Refresh), gotToken.TokenHash)
	assert.True(t, pair.RefreshExpiresAt.Equal(gotToken.ExpiresAt), "refresh expiry mismatch")
}

func TestLogin_NormalizesEmail(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(testUser(t, domain.ROLE_CLIENT), nil).Once()
	env.repo.On("AddRefreshToken", mock.Anything, mock.Anything).Return(nil).Once()

	_, _, err := env.service.Login(context.Background(), " User@Example.com ", TEST_PASSWORD)

	require.NoError(t, err)
}

func TestLogin_UnknownEmail(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(nil, domain.ErrUserNotFound).Once()

	_, _, err := env.service.Login(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLogin_WrongPassword(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(testUser(t, domain.ROLE_CLIENT), nil).Once()

	_, _, err := env.service.Login(context.Background(), TEST_EMAIL, TEST_WRONG_PASSWORD)

	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
	env.repo.AssertNotCalled(t, "AddRefreshToken", mock.Anything, mock.Anything)
}

func TestLogin_MalformedStoredHash(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	stored := testUser(t, domain.ROLE_CLIENT)
	stored.PasswordHash = "garbage"
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(stored, nil).Once()

	_, _, err := env.service.Login(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
	env.repo.AssertNotCalled(t, "AddRefreshToken", mock.Anything, mock.Anything)
}

func TestLogin_RepoError(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	repoErr := errors.New("db down")
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(nil, repoErr).Once()

	_, _, err := env.service.Login(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.ErrorIs(t, err, repoErr)
	assert.NotErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLogin_AddRefreshTokenFails(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	addErr := errors.New("insert failed")
	env.repo.On("GetUserByEmail", mock.Anything, TEST_EMAIL).Return(testUser(t, domain.ROLE_CLIENT), nil).Once()
	env.repo.On("AddRefreshToken", mock.Anything, mock.Anything).Return(addErr).Once()

	user, pair, err := env.service.Login(context.Background(), TEST_EMAIL, TEST_PASSWORD)

	require.ErrorIs(t, err, addErr)
	assert.Nil(t, user)
	assert.Nil(t, pair)
}

func TestLogin_EmptyEmail(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	_, _, err := env.service.Login(context.Background(), "  ", TEST_PASSWORD)

	require.ErrorIs(t, err, domain.ErrEmailRequired)
	env.repo.AssertNotCalled(t, "GetUserByEmail", mock.Anything, mock.Anything)
}

func TestLogin_EmptyPassword(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	_, _, err := env.service.Login(context.Background(), TEST_EMAIL, "")

	require.ErrorIs(t, err, domain.ErrPasswordRequired)
	env.repo.AssertNotCalled(t, "GetUserByEmail", mock.Anything, mock.Anything)
}

// --- Refresh ---

func TestRefresh_Success(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	user := testUser(t, domain.ROLE_CLIENT)
	var gotOldHash []byte
	var gotNext models.RefreshToken
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(activeRefreshToken(t, user.ID), nil).Once()
	env.repo.On("GetUserByID", mock.Anything, user.ID).Return(user, nil).Once()
	env.repo.On("RotateRefreshToken", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			gotOldHash = args.Get(1).([]byte)
			gotNext = args.Get(2).(models.RefreshToken)
		}).
		Return(nil).Once()

	pair, err := env.service.Refresh(context.Background(), TEST_REFRESH_TOKEN)

	require.NoError(t, err)
	require.NotNil(t, pair)
	assert.Equal(t, token.Hash(TEST_REFRESH_TOKEN), gotOldHash)
	assert.Equal(t, user.ID, gotNext.UserID)
	assert.Equal(t, token.Hash(pair.Refresh), gotNext.TokenHash)
	assert.True(t, pair.RefreshExpiresAt.Equal(gotNext.ExpiresAt), "refresh expiry mismatch")
	assert.NotEqual(t, TEST_REFRESH_TOKEN, pair.Refresh)
	env.repo.AssertNotCalled(t, "RevokeUserTokens", mock.Anything, mock.Anything)
}

func TestRefresh_Empty(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	_, err := env.service.Refresh(context.Background(), "")

	require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
	env.repo.AssertNotCalled(t, "GetRefreshToken", mock.Anything, mock.Anything)
}

func TestRefresh_NotFound(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(nil, domain.ErrRefreshTokenNotFound).Once()

	_, err := env.service.Refresh(context.Background(), TEST_REFRESH_TOKEN)

	require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
}

func TestRefresh_LookupError(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	lookupErr := errors.New("db down")
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(nil, lookupErr).Once()

	_, err := env.service.Refresh(context.Background(), TEST_REFRESH_TOKEN)

	require.ErrorIs(t, err, lookupErr)
}

func TestRefresh_Expired(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	userID := uuid.New()
	expired := storedRefreshToken(t, userID, time.Now().UTC().Add(-time.Hour), nil)
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(expired, nil).Once()

	_, err := env.service.Refresh(context.Background(), TEST_REFRESH_TOKEN)

	require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
	env.repo.AssertNotCalled(t, "RevokeUserTokens", mock.Anything, mock.Anything)
	env.repo.AssertNotCalled(t, "GetUserByID", mock.Anything, mock.Anything)
	env.repo.AssertNotCalled(t, "RotateRefreshToken", mock.Anything, mock.Anything, mock.Anything)
}

func TestRefresh_RevokedToken(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	userID := uuid.New()
	revokedAt := testTime
	revoked := storedRefreshToken(t, userID, time.Now().UTC().Add(time.Hour), &revokedAt)
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(revoked, nil).Once()
	env.repo.On("RevokeUserTokens", mock.Anything, userID).Return(nil).Once()

	_, err := env.service.Refresh(context.Background(), TEST_REFRESH_TOKEN)

	require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
	env.repo.AssertNotCalled(t, "GetUserByID", mock.Anything, mock.Anything)
	env.repo.AssertNotCalled(t, "RotateRefreshToken", mock.Anything, mock.Anything, mock.Anything)
}

func TestRefresh_RevokedAndExpired(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	userID := uuid.New()
	revokedAt := testTime
	revoked := storedRefreshToken(t, userID, time.Now().UTC().Add(-time.Hour), &revokedAt)
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(revoked, nil).Once()
	env.repo.On("RevokeUserTokens", mock.Anything, userID).Return(nil).Once()

	_, err := env.service.Refresh(context.Background(), TEST_REFRESH_TOKEN)

	require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
}

func TestRefresh_RevokeAllFails(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	userID := uuid.New()
	revokedAt := testTime
	revokeErr := errors.New("revoke failed")
	revoked := storedRefreshToken(t, userID, time.Now().UTC().Add(time.Hour), &revokedAt)
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(revoked, nil).Once()
	env.repo.On("RevokeUserTokens", mock.Anything, userID).Return(revokeErr).Once()

	_, err := env.service.Refresh(context.Background(), TEST_REFRESH_TOKEN)

	require.ErrorIs(t, err, revokeErr)
}

func TestRefresh_UserDeleted(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	userID := uuid.New()
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(activeRefreshToken(t, userID), nil).Once()
	env.repo.On("GetUserByID", mock.Anything, userID).Return(nil, domain.ErrUserNotFound).Once()

	_, err := env.service.Refresh(context.Background(), TEST_REFRESH_TOKEN)

	require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
	env.repo.AssertNotCalled(t, "RotateRefreshToken", mock.Anything, mock.Anything, mock.Anything)
}

func TestRefresh_UserLookupError(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	userID := uuid.New()
	lookupErr := errors.New("db down")
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(activeRefreshToken(t, userID), nil).Once()
	env.repo.On("GetUserByID", mock.Anything, userID).Return(nil, lookupErr).Once()

	_, err := env.service.Refresh(context.Background(), TEST_REFRESH_TOKEN)

	require.ErrorIs(t, err, lookupErr)
}

func TestRefresh_RotateRace(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	user := testUser(t, domain.ROLE_CLIENT)
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(activeRefreshToken(t, user.ID), nil).Once()
	env.repo.On("GetUserByID", mock.Anything, user.ID).Return(user, nil).Once()
	env.repo.On("RotateRefreshToken", mock.Anything, mock.Anything, mock.Anything).Return(domain.ErrRefreshTokenNotFound).Once()

	pair, err := env.service.Refresh(context.Background(), TEST_REFRESH_TOKEN)

	require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
	assert.Nil(t, pair)
}

func TestRefresh_RotateError(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	user := testUser(t, domain.ROLE_CLIENT)
	rotateErr := errors.New("tx failed")
	env.repo.On("GetRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(activeRefreshToken(t, user.ID), nil).Once()
	env.repo.On("GetUserByID", mock.Anything, user.ID).Return(user, nil).Once()
	env.repo.On("RotateRefreshToken", mock.Anything, mock.Anything, mock.Anything).Return(rotateErr).Once()

	pair, err := env.service.Refresh(context.Background(), TEST_REFRESH_TOKEN)

	require.ErrorIs(t, err, rotateErr)
	assert.Nil(t, pair)
}

// --- Logout ---

func TestLogout_Success(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("RevokeRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(nil).Once()

	require.NoError(t, env.service.Logout(context.Background(), TEST_REFRESH_TOKEN))
}

func TestLogout_Empty(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	err := env.service.Logout(context.Background(), "")

	require.ErrorIs(t, err, domain.ErrInvalidRefreshToken)
	env.repo.AssertNotCalled(t, "RevokeRefreshToken", mock.Anything, mock.Anything)
}

func TestLogout_UnknownTokenSucceeds(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	env.repo.On("RevokeRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(domain.ErrRefreshTokenNotFound).Once()

	require.NoError(t, env.service.Logout(context.Background(), TEST_REFRESH_TOKEN))
}

func TestLogout_RepoError(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	repoErr := errors.New("db down")
	env.repo.On("RevokeRefreshToken", mock.Anything, token.Hash(TEST_REFRESH_TOKEN)).Return(repoErr).Once()

	err := env.service.Logout(context.Background(), TEST_REFRESH_TOKEN)

	require.ErrorIs(t, err, repoErr)
}

// --- ValidateToken ---

func TestValidateToken_Empty(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	_, err := env.service.ValidateToken("")

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestValidateToken_Valid(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	user := testUser(t, domain.ROLE_OFFICER)
	pair, err := env.manager.NewPair(user)
	require.NoError(t, err)

	claims, err := env.service.ValidateToken(pair.Access)

	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Role, claims.Role)
}

func TestValidateToken_Garbage(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)

	_, err := env.service.ValidateToken("garbage")

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

// --- DeleteUser ---

func TestDeleteUser_Delegates(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	userID := uuid.New()
	env.repo.On("DeleteUser", mock.Anything, userID).Return(nil).Once()

	require.NoError(t, env.service.DeleteUser(context.Background(), userID))
}

func TestDeleteUser_Error(t *testing.T) {
	env := newTestEnv(t)
	defer env.assertExpectations(t)
	userID := uuid.New()
	repoErr := errors.New("db down")
	env.repo.On("DeleteUser", mock.Anything, userID).Return(repoErr).Once()

	err := env.service.DeleteUser(context.Background(), userID)

	require.ErrorIs(t, err, repoErr)
}
