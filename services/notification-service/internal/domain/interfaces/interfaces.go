package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

type NotificationRepository interface {
	// Save пишет event_id и уведомления одной транзакцией; повтор event_id - no-op и false.
	Save(ctx context.Context, envelope models.Envelope, notifications []models.Notification) (bool, error)
	ListByUser(ctx context.Context, userID uuid.UUID, page models.Page) ([]*models.Notification, error)
}

// Sender доставляет уведомление клиенту; сейчас это мок, который пишет в лог.
type Sender interface {
	Send(ctx context.Context, notification models.Notification) error
}

// EventReader читает топик группой консьюмера; оффсет двигается только явным Commit.
type EventReader interface {
	Fetch(ctx context.Context) (models.Message, error)
	Commit(ctx context.Context, msg models.Message) error
}

// DeadLetterWriter откладывает сообщение, которое не удалось обработать, в DLQ-топик.
type DeadLetterWriter interface {
	Send(ctx context.Context, msg models.Message, reason error) error
}
