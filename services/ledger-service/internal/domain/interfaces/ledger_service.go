package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

type LedgerService interface {
	Transfer(ctx context.Context, cmd models.TransferCommand) (*models.Transaction, error)
	Deposit(ctx context.Context, cmd models.DepositCommand) (*models.Transaction, error)
	GetTransaction(ctx context.Context, transactionID uuid.UUID) (*models.Transaction, error)
	ListTransactions(ctx context.Context, userID *uuid.UUID, accountID uuid.UUID, page models.Page) ([]*models.Transaction, error)
	GetBalance(ctx context.Context, accountID uuid.UUID) (*models.Balance, error)
}
