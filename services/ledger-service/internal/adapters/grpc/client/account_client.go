package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/account/v1"
)

var accountStatuses = map[accountv1.AccountStatus]models.AccountStatus{
	accountv1.AccountStatus_ACCOUNT_STATUS_ACTIVE:  models.ACCOUNT_STATUS_ACTIVE,
	accountv1.AccountStatus_ACCOUNT_STATUS_BLOCKED: models.ACCOUNT_STATUS_BLOCKED,
	accountv1.AccountStatus_ACCOUNT_STATUS_CLOSED:  models.ACCOUNT_STATUS_CLOSED,
}

type AccountClient struct {
	accounts accountv1.AccountServiceClient
}

func NewAccountClient(accounts accountv1.AccountServiceClient) *AccountClient {
	return &AccountClient{accounts: accounts}
}

func (c *AccountClient) GetAccount(ctx context.Context, id uuid.UUID) (*models.AccountInfo, error) {
	resp, err := c.accounts.GetAccount(ctx, &accountv1.GetAccountRequest{AccountId: id.String()})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("%w: %s", domain.ErrAccountNotFound, id)
		}

		return nil, fmt.Errorf("get account: %w", err)
	}

	userID, err := uuid.Parse(resp.UserId)
	if err != nil {
		return nil, fmt.Errorf("get account: user_id: %w", err)
	}

	return &models.AccountInfo{
		ID:       id,
		UserID:   userID,
		Currency: models.Currency(resp.Currency),
		Status:   accountStatuses[resp.Status],
	}, nil
}
