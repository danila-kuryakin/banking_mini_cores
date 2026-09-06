package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Auth *AuthRepo
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Auth: NewAuthRepo(db),
	}
}
