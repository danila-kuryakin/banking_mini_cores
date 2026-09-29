package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
)

type Customers interface {
	IsActive(ctx context.Context, userID uuid.UUID) (bool, error)
}

type Ledger interface {
	GetBalance(ctx context.Context, accountID uuid.UUID, currency models.Currency) (*models.Balance, error)
}
