package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/templates"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

type NotificationService struct {
	repo   interfaces.NotificationRepository
	sender interfaces.Sender
}

func NewNotificationService(repo interfaces.NotificationRepository, sender interfaces.Sender) *NotificationService {
	return &NotificationService{repo: repo, sender: sender}
}

// HandleEvent: разбор и шаблон - ошибка ErrPoisonMessage, запись в базу - обычная ошибка, которую стоит повторить.
func (s *NotificationService) HandleEvent(ctx context.Context, msg models.Message) error {
	envelope, err := models.ParseEnvelope(msg.Value)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrPoisonMessage, err)
	}

	notifications, err := templates.Build(envelope)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	for i := range notifications {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("notification id: %w", err)
		}

		notifications[i].ID = id
		notifications[i].CreatedAt = now
	}

	saved, err := s.repo.Save(ctx, envelope, notifications)
	if err != nil {
		return err
	}

	// Повторная доставка события уже отправленное не шлёт: отправка идёт только после первой записи.
	if !saved {
		return nil
	}

	for _, notification := range notifications {
		if err := s.sender.Send(ctx, notification); err != nil {
			return fmt.Errorf("send notification %s: %w", notification.ID, err)
		}
	}

	return nil
}

func (s *NotificationService) ListNotifications(ctx context.Context, userID uuid.UUID, page models.Page) ([]*models.Notification, error) {
	if page.Limit <= 0 {
		page.Limit = domain.DEFAULT_PAGE_SIZE
	} else if page.Limit > domain.MAX_PAGE_SIZE {
		page.Limit = domain.MAX_PAGE_SIZE
	}

	if page.Offset < 0 {
		page.Offset = 0
	}

	return s.repo.ListByUser(ctx, userID, page)
}
