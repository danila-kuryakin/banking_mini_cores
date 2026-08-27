package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type AntifraudRepo struct {
	db *pgxpool.Pool
}

func NewAntifraudRepo(db *pgxpool.Pool) *AntifraudRepo {
	return &AntifraudRepo{db: db}
}

func (r *AntifraudRepo) CheckTransfer() error {

	return nil
}
