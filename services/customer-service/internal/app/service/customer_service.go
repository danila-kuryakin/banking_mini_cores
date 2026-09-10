package service

import (
	"context"
	"fmt"
	"strings"
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

	return s.repo.Customer.UpdateProfile(ctx, userID, profile, nextStatus(profile))
}

func nextStatus(profile models.Profile) models.Status {
	if profile.IsComplete() {
		return models.STATUS_PROFILE_FILLED
	}

	return models.STATUS_NEW
}

// SetStatus переводит клиента в новый статус.
func (s *CustomerService) SetStatus(
	ctx context.Context,
	userID uuid.UUID,
	to models.Status,
	reason string,
	actorID *uuid.UUID,
) (*models.StatusChange, error) {
	if !to.IsValid() {
		return nil, domain.ErrStatusRequired
	}

	// TODO: Не забыть исправить в будущем
	// Блокировать клиента вправе только админ, а проверки роли здесь пока нет.
	// Таблица переходов blocked из любого статуса разрешает - закрыто именно
	// умение, а не переход.
	if to == models.STATUS_BLOCKED {
		return nil, domain.ErrBlockingNotImplemented
	}

	// Без причины запись в истории отвечает "когда отказали", но не "за что".
	if to == models.STATUS_REJECTED && strings.TrimSpace(reason) == "" {
		return nil, domain.ErrReasonRequired
	}

	current, changedAt, err := s.repo.Customer.GetStatusByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if *current == to {
		return &models.StatusChange{Previous: to, Current: to, ChangedAt: *changedAt}, nil
	}

	if !models.CanTransition(*current, to) {
		return nil, fmt.Errorf("%w: %s -> %s", domain.ErrStatusTransitionForbidden, *current, to)
	}

	return s.repo.Customer.SetStatus(ctx, userID, to, reason, actorID, models.AllowedFrom(to))
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
