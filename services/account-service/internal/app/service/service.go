package service

import (
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/adapters/grpc/client"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/interfaces"
)

type Service struct {
	Account interfaces.AccountService
}

func NewService(repo *repository.Repository, clients *client.Clients, cfg *config.Config) *Service {
	return &Service{
		Account: NewAccountService(repo, clients, cfg.CurrencySet, cfg.DefaultCurrencyCode, cfg.NumberGenerator),
	}
}
