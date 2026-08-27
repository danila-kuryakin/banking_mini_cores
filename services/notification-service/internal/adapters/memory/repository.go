package memory

import (
	repositoryApp "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/repository"
)

// Repository собирает хранилища сервиса - ровно как postgres.Repository в
// остальных сервисах, только за интерфейсом стоит память, а не база.
type Repository struct {
	Notification repositoryApp.Notification
}

func NewRepository() *Repository {
	return &Repository{
		Notification: NewNotificationRepo(),
	}
}
