package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomerRepo struct {
	db *pgxpool.Pool
}

func NewCustomerRepo(db *pgxpool.Pool) *CustomerRepo {
	return &CustomerRepo{db: db}
}

func (r *CustomerRepo) GetCustomer() error {

	return nil
}
