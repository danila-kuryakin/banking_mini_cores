package repository

import (
	repositoryApp "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/app/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Account repositoryApp.Account
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Account: NewAccountRepo(db),
	}
}
