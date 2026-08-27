package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type DocumentRepo struct {
	db *pgxpool.Pool
}

func NewDocumentRepo(db *pgxpool.Pool) *DocumentRepo {
	return &DocumentRepo{db: db}
}

func (r *DocumentRepo) ListDocuments() error {

	return nil
}
