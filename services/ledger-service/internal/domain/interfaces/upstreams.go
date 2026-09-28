package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

type Accounts interface {
	GetAccount(ctx context.Context, id uuid.UUID) (*models.AccountInfo, error)
}

// FraudChecker - антифрод; пока сервиса нет, стоит заглушка, которая всё разрешает.
type FraudChecker interface {
	CheckTransfer(ctx context.Context, cmd models.TransferCommand) (models.Verdict, error)
}

// EventPublisher отправляет события outbox в шину.
type EventPublisher interface {
	Publish(ctx context.Context, msgs []models.OutboxMessage) error
}
