package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
)

type MetadataRepository interface {
	// CreateMetadata сохраняет метаданные перед загрузкой файла.
	CreateMetadata(ctx context.Context, metadata models.Metadata) (*models.Metadata, error)
	// GetMetadata ищет запись по паре file_id + user_id.
	GetMetadata(ctx context.Context, fileID, userID uuid.UUID) (*models.Metadata, error)
	// ConfirmMetadata сохраняет метаданные после загрузки файла переводит и сохраняет статус confirmed.
	ConfirmMetadata(ctx context.Context, fileID, userID uuid.UUID, contentType string, sizeBytes int64, sha256 string) (*models.Metadata, error)
	// ListMetadata отдаёт файлы пользователя.
	ListMetadata(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Metadata, error)
	// ListConfirmedTypes отдаёт типы, которые пользователь уже загрузил.
	ListConfirmedTypes(ctx context.Context, userID uuid.UUID) ([]models.FileType, error)
}
