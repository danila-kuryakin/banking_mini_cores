package service

import (
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/grpc/client"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/repository"
)

type Service struct {
	Application *ApplicationService
	Sanctions   *SanctionsService
}

func NewService(repo *repository.Repository, clients *client.Clients) *Service {
	return &Service{
		Application: NewApplicationService(repo, clients),
		Sanctions:   NewSanctionsService(repo),
	}
}
