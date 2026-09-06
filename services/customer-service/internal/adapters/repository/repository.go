package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Customer *CustomerRepo
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Customer: NewCustomerRepo(db),
	}
}
