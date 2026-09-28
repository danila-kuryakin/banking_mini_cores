package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

type LedgerRepository interface {
	EnsureCashAccounts(ctx context.Context, currencies []models.Currency) (map[models.Currency]uuid.UUID, error)
	Post(ctx context.Context, p models.Posting) (*models.Transaction, error)
	Decline(ctx context.Context, p models.Posting, reason string) (*models.Transaction, error)
	GetTransaction(ctx context.Context, transactionId uuid.UUID) (*models.Transaction, error)
	ListTransactions(ctx context.Context, accountID uuid.UUID, page models.Page) ([]*models.Transaction, error)
	GetBalance(ctx context.Context, accountID uuid.UUID) (*models.Balance, error)
}

type OutboxRepository interface {
	// Relay забирает пачку неопубликованных событий, отдаёт их publish и помечает опубликованными.
	Relay(ctx context.Context, limit int, publish func(ctx context.Context, msgs []models.OutboxMessage) error) (int, error)
}
