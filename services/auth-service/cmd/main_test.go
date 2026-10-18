package main

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/mocks"
)

const (
	TEST_ADMIN_EMAIL    = "admin@example.com"
	TEST_ADMIN_PASSWORD = "admin-password"
)

// newTestService собирает сервис над моком репозитория.
func newTestService(t *testing.T, repo *mocks.AuthRepositoryMock) *service.Service {
	t.Helper()
	manager, err := token.NewManager("", 15*time.Minute, time.Hour, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	cfg := &config.Config{PasswordHashing: config.PasswordHashing{MaxConcurrent: 4, MaxWait: time.Second}}
	return service.NewService(&repository.Repository{Auth: repo}, manager, &mocks.CustomerProfilesMock{}, cfg)
}

func TestEnsureAdmin_NoPasswordSkips(t *testing.T) {
	repo := &mocks.AuthRepositoryMock{}
	defer repo.AssertExpectations(t)

	err := ensureAdmin(context.Background(), newTestService(t, repo), config.Admin{Email: TEST_ADMIN_EMAIL}, slog.New(slog.DiscardHandler))

	require.NoError(t, err)
	repo.AssertNotCalled(t, "GetUserByEmail", mock.Anything, mock.Anything)
}

func TestEnsureAdmin_AlreadyExists(t *testing.T) {
	repo := &mocks.AuthRepositoryMock{}
	defer repo.AssertExpectations(t)
	repo.On("GetUserByEmail", mock.Anything, TEST_ADMIN_EMAIL).
		Return(&models.User{ID: uuid.New(), Email: TEST_ADMIN_EMAIL, Role: domain.ROLE_ADMIN}, nil).Once()

	err := ensureAdmin(context.Background(), newTestService(t, repo), config.Admin{Email: TEST_ADMIN_EMAIL, Password: TEST_ADMIN_PASSWORD}, slog.New(slog.DiscardHandler))

	require.NoError(t, err)
}

func TestEnsureAdmin_ReturnsError(t *testing.T) {
	repo := &mocks.AuthRepositoryMock{}
	defer repo.AssertExpectations(t)
	repoErr := errors.New("db down")
	repo.On("GetUserByEmail", mock.Anything, TEST_ADMIN_EMAIL).Return(nil, repoErr).Once()

	err := ensureAdmin(context.Background(), newTestService(t, repo), config.Admin{Email: TEST_ADMIN_EMAIL, Password: TEST_ADMIN_PASSWORD}, slog.New(slog.DiscardHandler))

	assert.ErrorIs(t, err, repoErr)
}
