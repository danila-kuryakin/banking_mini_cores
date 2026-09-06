package service

import (
	"context"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/models"
	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
)

type CustomerService struct {
	repo *repository.Repository
}

func NewCustomerService(repo *repository.Repository) *CustomerService {
	return &CustomerService{repo: repo}
}

func (s *CustomerService) CreateProfile(ctx context.Context, userID uuid.UUID) (*models.Customer, error) {
	return s.repo.Customer.CreateCustomer(ctx, userID, models.STATUS_NEW)
}

func (s *CustomerService) GetCustomer(ctx context.Context, id uuid.UUID) (*models.Customer, error) {
	return s.repo.Customer.GetCustomer(ctx, id)
}

func (s *CustomerService) UpdateProfile(ctx context.Context, userID uuid.UUID, profile models.Profile) (*models.Customer, error) {
	status, _, err := s.repo.Customer.GetStatusByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if *status != models.STATUS_NEW { // что бы можно было заполнять только профиль со статусом NEW как в т.з.
		return nil, domain.ErrProfileLocked
	}

	return s.repo.Customer.UpdateProfile(ctx, userID, profile, models.STATUS_PROFILE_FILLED)
}

func (s *CustomerService) GetCustomerStatus(ctx context.Context, userID uuid.UUID) (*models.Status, *time.Time, error) {
	return s.repo.Customer.GetStatusByUserID(ctx, userID)
}

func (s *CustomerService) ListCustomers(ctx context.Context, limit, offset int) ([]*models.Customer, error) {
	if limit <= 0 {
		limit = domain.DEFAULT_PAGE_SIZE
	} else if limit > domain.MAX_PAGE_SIZE {
		limit = domain.MAX_PAGE_SIZE
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.Customer.ListCustomers(ctx, limit, offset)
}
