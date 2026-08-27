package repository

import (
	repositoryApp "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/app/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Kyc repositoryApp.Kyc
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Kyc: NewKycRepo(db),
	}
}
