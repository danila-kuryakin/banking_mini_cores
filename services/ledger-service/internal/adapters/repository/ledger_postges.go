package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type LedgerRepo struct {
	db *pgxpool.Pool
}

func NewLedgerRepo(db *pgxpool.Pool) *LedgerRepo {
	return &LedgerRepo{db: db}
}

func (r *LedgerRepo) GetTransaction() error {

	return nil
}
