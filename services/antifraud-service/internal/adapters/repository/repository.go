package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
)

type Repository struct {
	Rules  interfaces.RuleRepository
	Checks interfaces.CheckRepository
	Totals interfaces.TotalsRepository
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Rules:  NewRuleRepo(db),
		Checks: NewCheckRepo(db),
		Totals: NewTotalsRepo(db),
	}
}
