package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain/interfaces"
)

type Repository struct {
	Account interfaces.AccountRepository
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Account: NewAccountRepo(db),
	}
}
