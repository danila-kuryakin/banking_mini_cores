package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

// CustomerProfiles - customer-service глазами kyc.
type CustomerProfiles interface {
	// GetProfile отдаёт анкету и статус клиента.
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.CustomerProfile, error)
	// SetStatus двигает статус клиента по итогу заявки.
	SetStatus(ctx context.Context, userID uuid.UUID, to models.CustomerStatus, reason string, actorID *uuid.UUID) error
}

// Documents - filestore-service глазами kyc. Байты файлов сюда не ходят.
type Documents interface {
	// HasRequiredDocuments сообщает, залиты ли документы обязательных типов, и
	// каких не хватает.
	HasRequiredDocuments(ctx context.Context, userID uuid.UUID) (bool, []models.DocumentType, error)
	// ConfirmedTypes отдаёт типы загруженных документов клиента. Нужен для
	// повторной подачи.
	ConfirmedTypes(ctx context.Context, userID uuid.UUID) ([]models.DocumentType, error)
}
