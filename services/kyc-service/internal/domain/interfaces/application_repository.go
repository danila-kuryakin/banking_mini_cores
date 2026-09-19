package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

// ApplicationRepository владеет таблицами applications и kyc_events.
type ApplicationRepository interface {
	// Create заводит заявку и первую запись журнал событий одной транзакцией.
	Create(ctx context.Context, app models.Application) (*models.Application, error)
	// GetByID Возвращает заявки по ID.
	GetByID(ctx context.Context, id uuid.UUID) (*models.Application, error)
	// GetLatestByUserID Возвращает заявки по UserID.
	GetLatestByUserID(ctx context.Context, userID uuid.UUID) (*models.Application, error)
	// List Возвращает заявки после фильтрации, `ReviewerID` и пагинации, `Limit` и `Offset`.
	List(ctx context.Context, filter models.ApplicationFilter) ([]*models.Application, error)
	// Transition переводит заявку в новый статус и пишет запись журнал событий одной транзакцией.
	Transition(ctx context.Context, id uuid.UUID, t models.Transition, allowedFrom []models.ApplicationStatus) (*models.Application, error)
	// TakeForAutoChecks забирает пачку поданных заявок и сразу переводит их в
	// auto_checks, чтобы соседний воркер не взял те же самые.
	TakeForAutoChecks(ctx context.Context, limit int) ([]*models.Application, error)
	// Claim закрепляет заявку за офицером. Статус при этом не меняется, поэтому и
	// записи аудита здесь нет.
	Claim(ctx context.Context, id, officerID uuid.UUID) (*models.Application, error)
}
