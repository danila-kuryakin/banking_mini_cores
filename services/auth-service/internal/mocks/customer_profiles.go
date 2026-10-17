package mocks

import (
	"context"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/interfaces"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

var _ interfaces.CustomerProfiles = (*CustomerProfilesMock)(nil)

// CustomerProfilesMock - мок порта CustomerProfiles на testify/mock.
type CustomerProfilesMock struct {
	mock.Mock
}

func (m *CustomerProfilesMock) CreateProfile(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}
