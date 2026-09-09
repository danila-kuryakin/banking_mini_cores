package service

import (
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/adapters/storage_s3"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/interfaces"
)

type Service struct {
	Metadata interfaces.FileStorageService
}

func NewService(repo *repository.Repository, storage *storage_s3.Storage) *Service {
	return &Service{
		Metadata: NewFileStorageService(repo, storage),
	}
}
