package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
)

type AccountService interface {
	Open(ctx context.Context, userID uuid.UUID, rawCurrency string) (*models.Account, error)
	Get(ctx context.Context, userID *uuid.UUID, accountID uuid.UUID) (*models.Account, error)
	List(ctx context.Context, userID uuid.UUID, page models.Page) ([]*models.Account, error)
	Block(ctx context.Context, accountID, officerID uuid.UUID, reason string) (*models.Account, error)
	Unblock(ctx context.Context, accountID, officerID uuid.UUID, comment string) (*models.Account, error)
	Close(ctx context.Context, userID, accountID uuid.UUID) (*models.Account, error)
	Balance(ctx context.Context, userID, accountID uuid.UUID) (*models.Balance, error)
	History(ctx context.Context, accountID uuid.UUID, page models.Page) ([]*models.StatusChange, error)
}
