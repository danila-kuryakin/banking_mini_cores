package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/adapters/grpc/client"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
)

type AccountService struct {
	repo            *repository.Repository
	clients         *client.Clients
	currencies      models.CurrencySet
	defaultCurrency models.Currency
	numbers         *models.NumberGenerator
}

func NewAccountService(
	repo *repository.Repository,
	clients *client.Clients,
	currencies models.CurrencySet,
	defaultCurrency models.Currency,
	numbers *models.NumberGenerator,
) *AccountService {
	return &AccountService{
		repo:            repo,
		clients:         clients,
		currencies:      currencies,
		defaultCurrency: defaultCurrency,
		numbers:         numbers,
	}
}

func (s *AccountService) Open(ctx context.Context, userID uuid.UUID, rawCurrency string) (*models.Account, error) {
	currency := s.defaultCurrency

	if strings.TrimSpace(rawCurrency) != "" {
		parsed, err := models.ParseCurrency(rawCurrency)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", domain.ErrCurrencyNotAllowed, err)
		}

		currency = parsed
	}

	if !s.currencies.Contains(currency) {
		return nil, fmt.Errorf("%w: %s", domain.ErrCurrencyNotAllowed, currency)
	}

	active, err := s.clients.Customer.IsActive(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("open account: %w", err)
	}

	if !active {
		return nil, domain.ErrCustomerNotActive
	}

	serial, err := s.repo.Account.NextSerial(ctx)
	if err != nil {
		return nil, fmt.Errorf("open account: %w", err)
	}

	number, err := s.numbers.Build(currency, serial)
	if err != nil {
		return nil, fmt.Errorf("open account: %w", err)
	}

	accountID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("open account: generate account id: %w", err)
	}

	return s.repo.Account.Create(ctx, models.Account{
		ID:       accountID,
		UserID:   userID,
		Number:   number,
		Currency: currency,
	}, domain.MAX_ACCOUNTS_PER_CURRENCY)
}

func (s *AccountService) Get(ctx context.Context, userID *uuid.UUID, accountID uuid.UUID) (*models.Account, error) {
	return s.repo.Account.GetByID(ctx, userID, accountID)
}

// GetByNumber ищет счёт по номеру без владельца - так ходит ledger при переводе по номеру.
func (s *AccountService) GetByNumber(ctx context.Context, number string) (*models.Account, error) {
	return s.repo.Account.GetByNumber(ctx, number)
}

func (s *AccountService) List(ctx context.Context, userID uuid.UUID, page models.Page) ([]*models.Account, error) {
	if page.Limit <= 0 {
		page.Limit = domain.DEFAULT_PAGE_SIZE
	} else if page.Limit > domain.MAX_PAGE_SIZE {
		page.Limit = domain.MAX_PAGE_SIZE
	}

	if page.Offset < 0 {
		page.Offset = 0
	}

	return s.repo.Account.ListByUser(ctx, userID, page)
}

func (s *AccountService) History(ctx context.Context, accountID uuid.UUID, page models.Page) ([]*models.StatusChange, error) {
	if _, err := s.repo.Account.GetByID(ctx, nil, accountID); err != nil {
		return nil, err
	}

	if page.Limit <= 0 {
		page.Limit = domain.DEFAULT_PAGE_SIZE
	} else if page.Limit > domain.MAX_PAGE_SIZE {
		page.Limit = domain.MAX_PAGE_SIZE
	}

	if page.Offset < 0 {
		page.Offset = 0
	}

	return s.repo.Account.ListHistory(ctx, accountID, page)
}

func (s *AccountService) Block(ctx context.Context, accountID, officerID uuid.UUID, reason string) (*models.Account, error) {
	return s.transition(ctx, nil, accountID, models.Transition{To: models.STATUS_BLOCKED, ActorID: &officerID, Reason: reason})
}

func (s *AccountService) Unblock(ctx context.Context, accountID, officerID uuid.UUID, comment string) (*models.Account, error) {
	return s.transition(ctx, nil, accountID, models.Transition{To: models.STATUS_ACTIVE, ActorID: &officerID, Reason: comment})
}

func (s *AccountService) Close(ctx context.Context, userID, accountID uuid.UUID) (*models.Account, error) {
	account, err := s.repo.Account.GetByID(ctx, &userID, accountID)
	if err != nil {
		return nil, err
	}

	if account.Status == models.STATUS_CLOSED {
		return account, nil
	}

	if !models.CanTransition(account.Status, models.STATUS_CLOSED) {
		return nil, domain.ErrStatusConflict
	}

	balance, err := s.clients.Ledger.GetBalance(ctx, accountID, account.Currency)
	if err != nil {
		return nil, fmt.Errorf("close account: %w", err)
	}

	if balance.Amount != 0 {
		return nil, domain.ErrBalanceNotZero
	}

	return s.transition(ctx, &userID, accountID, models.Transition{To: models.STATUS_CLOSED})
}

func (s *AccountService) Balance(ctx context.Context, userID, accountID uuid.UUID) (*models.Balance, error) {
	account, err := s.repo.Account.GetByID(ctx, &userID, accountID)
	if err != nil {
		return nil, err
	}

	return s.clients.Ledger.GetBalance(ctx, accountID, account.Currency)
}

func (s *AccountService) transition(ctx context.Context, userID *uuid.UUID, accountID uuid.UUID, t models.Transition) (*models.Account, error) {
	account, err := s.repo.Account.GetByID(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}

	if account.Status == t.To {
		return account, nil
	}

	updated, err := s.repo.Account.Transition(ctx, accountID, t, models.AllowedFrom(t.To))
	if err != nil {
		if errors.Is(err, domain.ErrStatusConflict) || errors.Is(err, domain.ErrAccountNotFound) {
			return nil, err
		}

		return nil, fmt.Errorf("transition account: %w", err)
	}

	return updated, nil
}
