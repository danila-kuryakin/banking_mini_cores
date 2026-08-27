package repository

import (
	repositoryApp "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/app/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Ledger repositoryApp.Ledger
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Ledger: NewLedgerRepo(db),
	}
}
