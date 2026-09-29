package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/ledger/v1"
)

type LedgerClient struct {
	ledger ledgerv1.LedgerServiceClient
}

func NewLedgerClient(ledger ledgerv1.LedgerServiceClient) *LedgerClient {
	return &LedgerClient{ledger: ledger}
}

func (c *LedgerClient) GetBalance(ctx context.Context, accountID uuid.UUID, currency models.Currency) (*models.Balance, error) {
	resp, err := c.ledger.GetBalance(ctx, &ledgerv1.GetBalanceRequest{AccountId: accountID.String()})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return &models.Balance{AccountID: accountID, Currency: currency}, nil
		}

		return nil, fmt.Errorf("get balance: %w", err)
	}

	return &models.Balance{
		AccountID: accountID,
		Amount:    resp.Amount,
		Currency:  models.Currency(resp.Currency),
	}, nil
}
