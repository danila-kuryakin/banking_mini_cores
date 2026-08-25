package repository

import (
	repositoryApp "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/app/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Customer repositoryApp.Customer
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Customer: NewCustomerRepo(db),
	}
}
