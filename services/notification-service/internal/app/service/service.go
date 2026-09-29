package service

import (
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/interfaces"
)

type Service struct {
	Notifications *NotificationService
}

// NewService принимает весь конфиг, как остальные сервисы, хотя значений из него пока не раскладывает.
func NewService(repo *repository.Repository, sender interfaces.Sender, _ *config.Config) *Service {
	return &Service{
		Notifications: NewNotificationService(repo.Notifications, sender),
	}
}
