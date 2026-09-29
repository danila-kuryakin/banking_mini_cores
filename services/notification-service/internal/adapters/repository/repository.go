package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/interfaces"
)

type Repository struct {
	Notifications interfaces.NotificationRepository
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Notifications: NewNotificationRepo(db),
	}
}
