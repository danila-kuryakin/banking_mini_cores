package worker

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/interfaces"
)

// OutboxRelay публикует события из outbox в Kafka: at-least-once, дубли консьюмеры отсекают по event_id.
type OutboxRelay struct {
	outbox    interfaces.OutboxRepository
	publisher interfaces.EventPublisher
	log       *slog.Logger
	interval  time.Duration
	batchSize int
}

func NewOutboxRelay(
	outbox interfaces.OutboxRepository,
	publisher interfaces.EventPublisher,
	log *slog.Logger,
	interval time.Duration,
	batchSize int,
) *OutboxRelay {
	return &OutboxRelay{
		outbox:    outbox,
		publisher: publisher,
		log:       log,
		interval:  interval,
		batchSize: batchSize,
	}
}

// Run крутится до отмены контекста; ошибка пачки не роняет воркер, пачка уйдёт на следующем тике.
func (w *OutboxRelay) Run(ctx context.Context) {
	w.log.Info("outbox relay started", "interval", w.interval.String(), "batch_size", w.batchSize)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("outbox relay stopped")

			return
		case <-ticker.C:
			w.drain(ctx)
		}
	}
}

// drain публикует пачки подряд, пока полные: хвост после всплеска не ждёт по тику на пачку.
func (w *OutboxRelay) drain(ctx context.Context) {
	for {
		sent, err := w.outbox.Relay(ctx, w.batchSize, w.publisher.Publish)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				w.log.Error("failed to relay outbox", "error", err)
			}

			return
		}

		if sent > 0 {
			w.log.Debug("outbox events published", "count", sent)
		}

		if sent < w.batchSize {
			return
		}
	}
}
