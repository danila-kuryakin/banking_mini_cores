package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/models"
)

var _ interfaces.CustomerRepository = (*CustomerRepositoryMock)(nil)

// CustomerRepositoryMock - мок порта CustomerRepository на testify/mock.
type CustomerRepositoryMock struct {
	mock.Mock
}

func (m *CustomerRepositoryMock) CreateCustomer(ctx context.Context, userID uuid.UUID, status models.Status) (*models.Customer, error) {
	args := m.Called(ctx, userID, status)
	if v := args.Get(0); v != nil {
		return v.(*models.Customer), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *CustomerRepositoryMock) GetCustomer(ctx context.Context, userID uuid.UUID) (*models.Customer, error) {
	args := m.Called(ctx, userID)
	if v := args.Get(0); v != nil {
		return v.(*models.Customer), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *CustomerRepositoryMock) GetStatusByUserID(ctx context.Context, userID uuid.UUID) (*models.Status, *time.Time, error) {
	args := m.Called(ctx, userID)

	var customerStatus *models.Status
	if v := args.Get(0); v != nil {
		customerStatus = v.(*models.Status)
	}

	var changedAt *time.Time
	if v := args.Get(1); v != nil {
		changedAt = v.(*time.Time)
	}

	return customerStatus, changedAt, args.Error(2)
}

func (m *CustomerRepositoryMock) UpdateProfile(ctx context.Context, userID uuid.UUID, profile models.Profile, expected models.Status, next models.Status) (*models.Customer, error) {
	args := m.Called(ctx, userID, profile, expected, next)
	if v := args.Get(0); v != nil {
		return v.(*models.Customer), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *CustomerRepositoryMock) ListCustomers(ctx context.Context, limit, offset int) ([]*models.Customer, error) {
	args := m.Called(ctx, limit, offset)
	if v := args.Get(0); v != nil {
		return v.([]*models.Customer), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *CustomerRepositoryMock) SetStatus(ctx context.Context, userID uuid.UUID, to models.Status, reason string, actorID *uuid.UUID, allowedFrom []models.Status) (*models.StatusChange, error) {
	args := m.Called(ctx, userID, to, reason, actorID, allowedFrom)
	if v := args.Get(0); v != nil {
		return v.(*models.StatusChange), args.Error(1)
	}
	return nil, args.Error(1)
}
