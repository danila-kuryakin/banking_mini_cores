package interfaces

import (
	"context"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

type OutboxRepository interface {
	// Relay забирает пачку неопубликованных событий, отдаёт их publish и помечает опубликованными.
	Relay(ctx context.Context, limit int, publish func(ctx context.Context, msgs []models.OutboxMessage) error) (int, error)
}

// EventPublisher отправляет события outbox в шину.
type EventPublisher interface {
	Publish(ctx context.Context, msgs []models.OutboxMessage) error
}
