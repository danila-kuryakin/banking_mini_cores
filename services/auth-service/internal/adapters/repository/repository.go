package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/domain/interfaces"
)

type Repository struct {
	Auth          interfaces.AuthRepository
	LoginAttempts interfaces.LoginAttemptRepository
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Auth:          NewAuthRepo(db),
		LoginAttempts: NewLoginAttemptRepo(db),
	}
}
