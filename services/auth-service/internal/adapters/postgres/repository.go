package postgres

import (
	repositoryApp "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Auth repositoryApp.Auth
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Auth: NewAuthRepo(db),
	}
}
