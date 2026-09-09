package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
)

type FileStorageService interface {
	// InitUpload сохраняет метаданные и выдаёт ссылку куда загружать файл в хранилище.
	InitUpload(ctx context.Context, userID uuid.UUID, fileType models.FileType, filename string) (*models.Metadata, *models.PresignedURL, error)
	// ConfirmUpload проверяет, что файл загружен, и фиксирует метаданные.
	ConfirmUpload(ctx context.Context, userID, fileID uuid.UUID) (*models.Metadata, error)
	// GetDownloadURL отдаёт ссылку на скачивание.
	GetDownloadURL(ctx context.Context, userID, fileID uuid.UUID) (*models.PresignedURL, error)
	// ListFiles Список файлов из базы данных с пагинацией
	ListFiles(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Metadata, error)
	// HasRequiredFiles отвечает kyc-service, можно ли принимать заявку.
	HasRequiredFiles(ctx context.Context, userID uuid.UUID) (bool, []models.FileType, error)
}
