package mocks

import (
	"context"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

var _ interfaces.AuthRepository = (*AuthRepositoryMock)(nil)

// AuthRepositoryMock - мок порта AuthRepository на testify/mock.
type AuthRepositoryMock struct {
	mock.Mock
}

func (m *AuthRepositoryMock) CreateUser(ctx context.Context, user models.User) (*models.User, error) {
	args := m.Called(ctx, user)
	if v := args.Get(0); v != nil {
		return v.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *AuthRepositoryMock) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *AuthRepositoryMock) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if v := args.Get(0); v != nil {
		return v.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *AuthRepositoryMock) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, userID)
	if v := args.Get(0); v != nil {
		return v.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *AuthRepositoryMock) AddRefreshToken(ctx context.Context, token models.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *AuthRepositoryMock) GetRefreshToken(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
	args := m.Called(ctx, hash)
	if v := args.Get(0); v != nil {
		return v.(*models.RefreshToken), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *AuthRepositoryMock) RevokeRefreshToken(ctx context.Context, hash []byte) error {
	args := m.Called(ctx, hash)
	return args.Error(0)
}

func (m *AuthRepositoryMock) RevokeUserTokens(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *AuthRepositoryMock) RotateRefreshToken(ctx context.Context, oldHash []byte, next models.RefreshToken) error {
	args := m.Called(ctx, oldHash, next)
	return args.Error(0)
}
