package service

import (
	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/adapters/grpc/client"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

type Service struct {
	Ledger *LedgerService
}

// NewService раскладывает значения конфига по сервисам; кассовые счета приходят из БД, а не из конфига.
func NewService(
	repo *repository.Repository,
	clients *client.Clients,
	cfg *config.Config,
	cash map[models.Currency]uuid.UUID,
) *Service {
	return &Service{
		Ledger: NewLedgerService(repo.Ledger, clients.Accounts, clients.Fraud, cfg.CurrencySet, cash),
	}
}
