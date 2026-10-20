package mocks

import (
	"context"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/models"
	"github.com/stretchr/testify/mock"
)

var _ interfaces.LoginAttemptRepository = (*LoginAttemptRepositoryMock)(nil)

// LoginAttemptRepositoryMock - мок порта LoginAttemptRepository на testify/mock.
type LoginAttemptRepositoryMock struct {
	mock.Mock
}

func (m *LoginAttemptRepositoryMock) LockedUntil(ctx context.Context, email string, now time.Time) (*time.Time, error) {
	args := m.Called(ctx, email, now)
	if v := args.Get(0); v != nil {
		return v.(*time.Time), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *LoginAttemptRepositoryMock) RegisterFailure(ctx context.Context, email string, failure models.LoginFailure) error {
	args := m.Called(ctx, email, failure)
	return args.Error(0)
}

func (m *LoginAttemptRepositoryMock) Reset(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}
