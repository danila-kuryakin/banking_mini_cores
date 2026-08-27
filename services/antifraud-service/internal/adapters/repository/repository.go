package repository

import (
	repositoryApp "github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/app/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Antifraud repositoryApp.Antifraud
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Antifraud: NewAntifraudRepo(db),
	}
}
