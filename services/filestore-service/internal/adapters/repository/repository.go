package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/interfaces"
)

type Repository struct {
	Metadata interfaces.MetadataRepository
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Metadata: NewMetadataRepo(db),
	}
}
