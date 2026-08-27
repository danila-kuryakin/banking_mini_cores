package repository

import (
	repositoryApp "github.com/danila-kuryakin/banking_mini_cores/services/document-service/internal/app/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Document repositoryApp.Document
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Document: NewDocumentRepo(db),
	}
}
