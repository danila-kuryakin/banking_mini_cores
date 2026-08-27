package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type KycRepo struct {
	db *pgxpool.Pool
}

func NewKycRepo(db *pgxpool.Pool) *KycRepo {
	return &KycRepo{db: db}
}

func (r *KycRepo) GetApplication() error {

	return nil
}
