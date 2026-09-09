// Package filerepo - реализация порта MinIORepository поверх minio-go.
//
// Имя пакета намеренно не minio: SDK импортируется здесь под этим же именем,
// и одноимённый локальный пакет читался бы как ошибка.
package filerepo

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
)

type FileRepository struct {
	client     *minio.Client
	bucketName string
}

func NewFileRepository(client *minio.Client, bucketName string) *FileRepository {
	return &FileRepository{client: client, bucketName: bucketName}
}

// PresignedPutURL выдаёт ссылку, по которой клиент загрузит файл напрямую в хранилище.
func (r *FileRepository) PresignedPutURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error) {
	presignedURL, err := r.client.PresignedPutObject(ctx, r.bucketName, objectPath, expiry)
	if err != nil {
		return "", fmt.Errorf("presign put %s: %w", objectPath, err)
	}

	return presignedURL.String(), nil
}

// PresignedGetURL выдаёт ссылку на скачивание.
func (r *FileRepository) PresignedGetURL(ctx context.Context, objectPath string, expiry time.Duration) (string, error) {
	presignedURL, err := r.client.PresignedGetObject(ctx, r.bucketName, objectPath, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("presign get %s: %w", objectPath, err)
	}

	return presignedURL.String(), nil
}

// StatObject проверяет, что файл действительно загружен, и отдаёт его размер.
func (r *FileRepository) StatObject(ctx context.Context, objectPath string) (models.ObjectInfo, error) {
	objInfo, err := r.client.StatObject(ctx, r.bucketName, objectPath, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == domain.NO_SUCH_KEY_CODE {
			return models.ObjectInfo{}, domain.ErrObjectNotFound
		}

		return models.ObjectInfo{}, fmt.Errorf("stat object %s: %w", objectPath, err)
	}

	return models.ObjectInfo{
		Key:          objInfo.Key,
		Size:         objInfo.Size,
		ContentType:  objInfo.ContentType,
		LastModified: objInfo.LastModified,
	}, nil
}

// GetObject отдаёт файл.
func (r *FileRepository) GetObject(ctx context.Context, objectPath string) (io.ReadCloser, error) {
	object, err := r.client.GetObject(ctx, r.bucketName, objectPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", objectPath, err)
	}

	if _, err := object.Stat(); err != nil {
		_ = object.Close()

		if minio.ToErrorResponse(err).Code == domain.NO_SUCH_KEY_CODE {
			return nil, domain.ErrObjectNotFound
		}

		return nil, fmt.Errorf("get object %s: %w", objectPath, err)
	}

	return object, nil
}

// RemoveObject удаляет файл.
func (r *FileRepository) RemoveObject(ctx context.Context, objectPath string) error {
	if err := r.client.RemoveObject(ctx, r.bucketName, objectPath, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object %s: %w", objectPath, err)
	}

	return nil
}
