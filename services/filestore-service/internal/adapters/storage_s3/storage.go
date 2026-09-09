package storage_s3

import (
	"github.com/minio/minio-go/v7"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/adapters/storage_s3/filerepo"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/interfaces"
)

type Storage struct {
	MinIO interfaces.MinIORepository
}

func NewStorage(client *minio.Client, bucketName string) *Storage {
	return &Storage{
		MinIO: filerepo.NewFileRepository(client, bucketName),
	}
}
