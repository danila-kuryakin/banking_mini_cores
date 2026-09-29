package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/interfaces"
)

type Repository struct {
	Application interfaces.ApplicationRepository
	Check       interfaces.CheckRepository
	Sanctions   interfaces.SanctionsRepository
	Outbox      interfaces.OutboxRepository
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Application: NewApplicationRepo(db),
		Check:       NewCheckRepo(db),
		Sanctions:   NewSanctionsRepo(db),
		Outbox:      NewOutboxRepo(db),
	}
}
