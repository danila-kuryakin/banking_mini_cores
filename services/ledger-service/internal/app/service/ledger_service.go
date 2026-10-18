package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

type LedgerService struct {
	repo       interfaces.LedgerRepository
	accounts   interfaces.Accounts
	fraud      interfaces.FraudChecker
	currencies models.CurrencySet
	cash       map[models.Currency]uuid.UUID
}

func NewLedgerService(
	repo interfaces.LedgerRepository,
	accounts interfaces.Accounts,
	fraud interfaces.FraudChecker,
	currencies models.CurrencySet,
	cash map[models.Currency]uuid.UUID,
) *LedgerService {
	return &LedgerService{
		repo:       repo,
		accounts:   accounts,
		fraud:      fraud,
		currencies: currencies,
		cash:       cash,
	}
}

func (s *LedgerService) Transfer(ctx context.Context, cmd models.TransferCommand) (*models.Transaction, error) {
	currency, err := s.currency(cmd.Currency)
	if err != nil {
		return nil, err
	}

	if cmd.From == cmd.To {
		return nil, domain.ErrSameAccount
	}

	from, err := s.activeAccount(ctx, cmd.From, currency)
	if err != nil {
		return nil, err
	}

	if from.UserID != cmd.InitiatorID {
		return nil, domain.ErrAccountNotOwned
	}

	to, err := s.activeAccount(ctx, cmd.To, currency)
	if err != nil {
		return nil, err
	}

	posting := models.Posting{
		Type:        models.TYPE_TRANSFER,
		From:        cmd.From,
		To:          cmd.To,
		Amount:      cmd.Amount,
		Currency:    currency,
		InitiatorID: &cmd.InitiatorID,
		FromUserID:  &from.UserID,
		ToUserID:    &to.UserID,
	}

	verdict, err := s.fraud.CheckTransfer(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("transfer: %w: %v", domain.ErrFraudCheckFailed, err)
	}

	if !verdict.Allowed {
		reason := domain.DECLINE_ANTIFRAUD
		if verdict.CheckID != "" {
			reason += " (check " + verdict.CheckID + ")"
		}

		if len(verdict.Reasons) > 0 {
			reason += ": " + strings.Join(verdict.Reasons, ", ")
		}

		declined, err := s.repo.Decline(ctx, posting, reason)
		if err != nil {
			return nil, fmt.Errorf("transfer: %w", err)
		}

		return declined, domain.ErrTransferDenied
	}

	return s.repo.Post(ctx, posting)
}

// Deposit - пополнение из кассы: дебет кассового счёта валюты, кредит счёта клиента.
func (s *LedgerService) Deposit(ctx context.Context, cmd models.DepositCommand) (*models.Transaction, error) {
	currency, err := s.currency(cmd.Currency)
	if err != nil {
		return nil, err
	}

	account, err := s.activeAccount(ctx, cmd.AccountID, currency)
	if err != nil {
		return nil, err
	}

	cash, ok := s.cash[currency]
	if !ok {
		return nil, fmt.Errorf("deposit: no cash account for %s", currency)
	}

	return s.repo.Post(ctx, models.Posting{
		Type:        models.TYPE_DEPOSIT,
		From:        cash,
		To:          cmd.AccountID,
		Amount:      cmd.Amount,
		Currency:    currency,
		InitiatorID: &cmd.OfficerID,
		ToUserID:    &account.UserID,
	})
}

func (s *LedgerService) GetTransaction(ctx context.Context, transactionID uuid.UUID) (*models.Transaction, error) {
	return s.repo.GetTransaction(ctx, transactionID)
}

// ListTransactions - выписка; непустой userID ограничивает её счетами этого клиента.
func (s *LedgerService) ListTransactions(
	ctx context.Context,
	userID *uuid.UUID,
	accountID uuid.UUID,
	page models.Page,
) ([]*models.Transaction, error) {
	if userID != nil {
		account, err := s.accounts.GetAccount(ctx, accountID)
		if err != nil {
			return nil, err
		}

		if account.UserID != *userID {
			return nil, domain.ErrAccountNotFound
		}
	}

	if page.Limit <= 0 {
		page.Limit = domain.DEFAULT_PAGE_SIZE
	} else if page.Limit > domain.MAX_PAGE_SIZE {
		page.Limit = domain.MAX_PAGE_SIZE
	}

	if page.Offset < 0 {
		page.Offset = 0
	}

	return s.repo.ListTransactions(ctx, accountID, page)
}

func (s *LedgerService) GetBalance(ctx context.Context, accountID uuid.UUID) (*models.Balance, error) {
	return s.repo.GetBalance(ctx, accountID)
}

func (s *LedgerService) currency(raw string) (models.Currency, error) {
	currency, err := models.ParseCurrency(raw)
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrCurrencyNotAllowed, err)
	}

	if !s.currencies.Contains(currency) {
		return "", fmt.Errorf("%w: %s", domain.ErrCurrencyNotAllowed, currency)
	}

	return currency, nil
}

func (s *LedgerService) activeAccount(ctx context.Context, id uuid.UUID, currency models.Currency) (*models.AccountInfo, error) {
	account, err := s.accounts.GetAccount(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			return nil, err
		}

		return nil, fmt.Errorf("get account %s: %w", id, err)
	}

	if account.Status != models.ACCOUNT_STATUS_ACTIVE {
		return nil, fmt.Errorf("%w: %s is %s", domain.ErrAccountNotActive, id, account.Status)
	}

	if account.Currency != currency {
		return nil, fmt.Errorf("%w: %s is in %s", domain.ErrCurrencyMismatch, id, account.Currency)
	}

	return account, nil
}
