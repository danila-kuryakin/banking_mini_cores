package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain/interfaces"
)

type Repository struct {
	Customer interfaces.CustomerRepository
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Customer: NewCustomerRepo(db),
	}
}
