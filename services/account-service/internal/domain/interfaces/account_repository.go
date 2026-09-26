package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/models"
)

type AccountRepository interface {
	NextSerial(ctx context.Context) (int64, error)
	Create(ctx context.Context, account models.Account, maxPerCurrency int) (*models.Account, error)
	GetByID(ctx context.Context, userID *uuid.UUID, accountID uuid.UUID) (*models.Account, error)
	ListByUser(ctx context.Context, userID uuid.UUID, page models.Page) ([]*models.Account, error)
	ListHistory(ctx context.Context, accountID uuid.UUID, page models.Page) ([]*models.StatusChange, error)
	Transition(ctx context.Context, accountID uuid.UUID, t models.Transition, allowedFrom []models.AccountStatus) (*models.Account, error)
}
