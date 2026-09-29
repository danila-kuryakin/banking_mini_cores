package service

import (
	"context"
	"time"

	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/account/v1"
)

type AccountService struct {
	accountClient accountv1.AccountServiceClient
	timeout       time.Duration
}

func NewAccountService(accountCli accountv1.AccountServiceClient, timeout time.Duration) *AccountService {
	return &AccountService{
		accountClient: accountCli,
		timeout:       timeout,
	}
}

func (s AccountService) Open(ctx context.Context, userID, currency string) (*accountv1.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.accountClient.OpenAccount(ctx, &accountv1.OpenAccountRequest{UserId: userID, Currency: currency})
}

func (s AccountService) Get(ctx context.Context, userID, accountID string) (*accountv1.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.accountClient.GetAccount(ctx, &accountv1.GetAccountRequest{UserId: userID, AccountId: accountID})
}

func (s AccountService) List(ctx context.Context, in *accountv1.ListAccountsRequest) ([]*accountv1.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	listed, err := s.accountClient.ListAccounts(ctx, in)
	if err != nil {
		return nil, err
	}

	return listed.Accounts, nil
}

func (s AccountService) Block(ctx context.Context, accountID, officerID, reason string) (*accountv1.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.accountClient.BlockAccount(ctx, &accountv1.BlockAccountRequest{
		AccountId: accountID,
		OfficerId: officerID,
		Reason:    reason,
	})
}

func (s AccountService) Unblock(ctx context.Context, accountID, officerID, comment string) (*accountv1.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.accountClient.UnblockAccount(ctx, &accountv1.UnblockAccountRequest{
		AccountId: accountID,
		OfficerId: officerID,
		Comment:   comment,
	})
}

func (s AccountService) Balance(ctx context.Context, userID, accountID string) (*accountv1.GetBalanceResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.accountClient.GetBalance(ctx, &accountv1.GetBalanceRequest{UserId: userID, AccountId: accountID})
}

func (s AccountService) Close(ctx context.Context, userID, accountID string) (*accountv1.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.accountClient.CloseAccount(ctx, &accountv1.CloseAccountRequest{UserId: userID, AccountId: accountID})
}

func (s AccountService) History(ctx context.Context, in *accountv1.ListStatusHistoryRequest) ([]*accountv1.StatusChange, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	listed, err := s.accountClient.ListStatusHistory(ctx, in)
	if err != nil {
		return nil, err
	}

	return listed.Changes, nil
}
