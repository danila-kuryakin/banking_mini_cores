package interfaces

import (
	"context"
	"io"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
)

type MinIORepository interface {
	// PresignedPutURL выдаёт ссылку, по которой клиент загрузит файл напрямую в хранилище.
	PresignedPutURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error)
	// PresignedGetURL выдаёт ссылку на скачивание.
	PresignedGetURL(ctx context.Context, objectPath string, expiry time.Duration, opts models.DownloadOptions) (string, error)
	// StatObject проверяет, что файл действительно загружен, и отдаёт его размер.
	StatObject(ctx context.Context, objectPath string) (models.ObjectInfo, error)
	// GetObject отдаёт файл.
	GetObject(ctx context.Context, objectPath string) (io.ReadCloser, error)
	// RemoveObject удаляет файл.
	RemoveObject(ctx context.Context, objectPath string) error
}
