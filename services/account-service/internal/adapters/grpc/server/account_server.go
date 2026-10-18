package server

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/account/v1"
)

type AccountServer struct {
	accountv1.UnimplementedAccountServiceServer
	service *service.Service
	log     *slog.Logger
}

func NewAccountServer(service *service.Service, log *slog.Logger) *AccountServer {
	return &AccountServer{service: service, log: log}
}

func (s *AccountServer) OpenAccount(ctx context.Context, in *accountv1.OpenAccountRequest) (*accountv1.Account, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrUserIDMustBeUUID
	}

	account, err := s.service.Account.Open(ctx, userID, in.Currency)
	if err != nil {
		return nil, s.fail("failed to open account", err, domain.ErrOpenFailed, "user_id", in.UserId)
	}

	return accountToProto(account), nil
}

func (s *AccountServer) GetAccount(ctx context.Context, in *accountv1.GetAccountRequest) (*accountv1.Account, error) {
	var userID *uuid.UUID

	if in.UserId != "" {
		parsed, err := uuid.Parse(in.UserId)
		if err != nil {
			return nil, domain.ErrUserIDMustBeUUID
		}

		userID = &parsed
	}

	accountID, err := uuid.Parse(in.AccountId)
	if err != nil {
		return nil, domain.ErrAccountIDMustBeUUID
	}

	account, err := s.service.Account.Get(ctx, userID, accountID)
	if err != nil {
		return nil, s.fail("failed to get account", err, domain.ErrGetFailed, "account_id", in.AccountId)
	}

	return accountToProto(account), nil
}

func (s *AccountServer) ListAccounts(ctx context.Context, in *accountv1.ListAccountsRequest) (*accountv1.ListAccountsResponse, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrUserIDMustBeUUID
	}

	accounts, err := s.service.Account.List(ctx, userID, models.Page{
		Limit:  int(in.Limit),
		Offset: int(in.Offset),
	})
	if err != nil {
		return nil, s.fail("failed to list accounts", err, domain.ErrListFailed, "user_id", in.UserId)
	}

	return &accountv1.ListAccountsResponse{Accounts: accountsToProto(accounts)}, nil
}

func (s *AccountServer) BlockAccount(ctx context.Context, in *accountv1.BlockAccountRequest) (*accountv1.Account, error) {
	accountID, err := uuid.Parse(in.AccountId)
	if err != nil {
		return nil, domain.ErrAccountIDMustBeUUID
	}

	officerID, err := uuid.Parse(in.OfficerId)
	if err != nil {
		return nil, domain.ErrOfficerIDMustBeUUID
	}

	account, err := s.service.Account.Block(ctx, accountID, officerID, strings.TrimSpace(in.Reason))
	if err != nil {
		return nil, s.fail("failed to block account", err, domain.ErrUpdateFailed, "account_id", in.AccountId)
	}

	return accountToProto(account), nil
}

func (s *AccountServer) UnblockAccount(ctx context.Context, in *accountv1.UnblockAccountRequest) (*accountv1.Account, error) {
	accountID, err := uuid.Parse(in.AccountId)
	if err != nil {
		return nil, domain.ErrAccountIDMustBeUUID
	}

	officerID, err := uuid.Parse(in.OfficerId)
	if err != nil {
		return nil, domain.ErrOfficerIDMustBeUUID
	}

	account, err := s.service.Account.Unblock(ctx, accountID, officerID, strings.TrimSpace(in.Comment))
	if err != nil {
		return nil, s.fail("failed to unblock account", err, domain.ErrUpdateFailed, "account_id", in.AccountId)
	}

	return accountToProto(account), nil
}

func (s *AccountServer) CloseAccount(ctx context.Context, in *accountv1.CloseAccountRequest) (*accountv1.Account, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrUserIDMustBeUUID
	}

	accountID, err := uuid.Parse(in.AccountId)
	if err != nil {
		return nil, domain.ErrAccountIDMustBeUUID
	}

	account, err := s.service.Account.Close(ctx, userID, accountID)
	if err != nil {
		return nil, s.fail("failed to close account", err, domain.ErrUpdateFailed, "account_id", in.AccountId)
	}

	return accountToProto(account), nil
}

func (s *AccountServer) GetBalance(ctx context.Context, in *accountv1.GetBalanceRequest) (*accountv1.GetBalanceResponse, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrUserIDMustBeUUID
	}

	accountID, err := uuid.Parse(in.AccountId)
	if err != nil {
		return nil, domain.ErrAccountIDMustBeUUID
	}

	balance, err := s.service.Account.Balance(ctx, userID, accountID)
	if err != nil {
		return nil, s.fail("failed to get balance", err, domain.ErrBalanceFailed, "account_id", in.AccountId)
	}

	return &accountv1.GetBalanceResponse{
		AccountId: balance.AccountID.String(),
		Amount:    balance.Amount,
		Currency:  string(balance.Currency),
	}, nil
}

func (s *AccountServer) ListStatusHistory(ctx context.Context, in *accountv1.ListStatusHistoryRequest) (*accountv1.ListStatusHistoryResponse, error) {
	accountID, err := uuid.Parse(in.AccountId)
	if err != nil {
		return nil, domain.ErrAccountIDMustBeUUID
	}

	changes, err := s.service.Account.History(ctx, accountID, models.Page{
		Limit:  int(in.Limit),
		Offset: int(in.Offset),
	})
	if err != nil {
		return nil, s.fail("failed to list status history", err, domain.ErrHistoryFailed, "account_id", in.AccountId)
	}

	return &accountv1.ListStatusHistoryResponse{Changes: statusChangesToProto(changes)}, nil
}

// fail переводит доменную ошибку в gRPC-статус для логирования.
func (s *AccountServer) fail(msg string, err error, fallback error, logArgs ...any) error {
	switch {
	case errors.Is(err, domain.ErrAccountNotFound):
		return domain.ErrAccountMissing
	case errors.Is(err, domain.ErrCustomerNotFound):
		return domain.ErrCustomerMissing
	case errors.Is(err, domain.ErrCustomerNotActive):
		return domain.ErrCustomerIsNotActive
	case errors.Is(err, domain.ErrCurrencyNotAllowed):
		s.log.Error(msg, append(logArgs, "error", err)...)
		return domain.ErrCurrencyIsNotValid
	case errors.Is(err, domain.ErrAccountLimitReached):
		return domain.ErrAccountLimit
	case errors.Is(err, domain.ErrStatusConflict):
		return domain.ErrTransitionForbidden
	case errors.Is(err, domain.ErrBalanceNotZero):
		return domain.ErrBalanceIsNotZero
	}

	if _, isStatus := status.FromError(err); isStatus {
		return err
	}

	s.log.Error(msg, append(logArgs, "error", err)...)

	return fallback
}
