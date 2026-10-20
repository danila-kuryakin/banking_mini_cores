package mocks

import (
	"context"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/interfaces"
	"github.com/stretchr/testify/mock"
)

var _ interfaces.PasswordHasher = (*PasswordHasherMock)(nil)

// PasswordHasherMock - мок порта PasswordHasher на testify/mock.
type PasswordHasherMock struct {
	mock.Mock
}

func (m *PasswordHasherMock) Hash(ctx context.Context, plain string) (string, error) {
	args := m.Called(ctx, plain)
	return args.String(0), args.Error(1)
}

func (m *PasswordHasherMock) Verify(ctx context.Context, encoded, plain string) error {
	args := m.Called(ctx, encoded, plain)
	return args.Error(0)
}
