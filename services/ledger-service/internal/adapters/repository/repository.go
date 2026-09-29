package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/interfaces"
)

type Repository struct {
	Ledger interfaces.LedgerRepository
	Outbox interfaces.OutboxRepository
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Ledger: NewLedgerRepo(db),
		Outbox: NewOutboxRepo(db),
	}
}
