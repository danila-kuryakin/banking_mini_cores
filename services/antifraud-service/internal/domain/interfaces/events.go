package interfaces

import (
	"context"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

// EventReader читает топик группой консьюмера; оффсет двигается только явным Commit.
type EventReader interface {
	Fetch(ctx context.Context) (models.Message, error)
	Commit(ctx context.Context, msg models.Message) error
}
