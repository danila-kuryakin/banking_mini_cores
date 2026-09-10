package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/models"
)

type CustomerRepository interface {
	// CreateCustomer заводит карточку клиента и первую запись в истории статусов.
	CreateCustomer(ctx context.Context, userID uuid.UUID, status models.Status) (*models.Customer, error)
	// GetCustomer отдаёт карточку вместе с анкетой.
	GetCustomer(ctx context.Context, userID uuid.UUID) (*models.Customer, error)
	// GetStatusByUserID отдаёт только статус и время его последней смены.
	GetStatusByUserID(ctx context.Context, userID uuid.UUID) (*models.Status, *time.Time, error)
	// UpdateProfile сохраняет анкету; статус меняется вместе с ней.
	UpdateProfile(ctx context.Context, userID uuid.UUID, profile models.Profile, status models.Status) (*models.Customer, error)
	// ListCustomers отдаёт карточки постранично.
	ListCustomers(ctx context.Context, limit, offset int) ([]*models.Customer, error)
	// SetStatus переводит клиента в статус to и пишет переход в историю.
	SetStatus(ctx context.Context, userID uuid.UUID, to models.Status, reason string, actorID *uuid.UUID, allowedFrom []models.Status) (*models.StatusChange, error)
}
