package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

type OutboxRepo struct {
	db *pgxpool.Pool
}

func NewOutboxRepo(db *pgxpool.Pool) *OutboxRepo {
	return &OutboxRepo{db: db}
}

// Relay держит строки заблокированными, пока идёт запись в Kafka; ошибка publish откатывает пометку, и пачка уйдёт повторно.
func (r *OutboxRepo) Relay(
	ctx context.Context,
	limit int,
	publish func(ctx context.Context, msgs []models.OutboxMessage) error,
) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("relay outbox: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, TAKE_OUTBOX_QUERY, limit)
	if err != nil {
		return 0, fmt.Errorf("relay outbox: take: %w", err)
	}

	msgs := make([]models.OutboxMessage, 0, limit)
	ids := make([]uuid.UUID, 0, limit)

	for rows.Next() {
		var msg models.OutboxMessage

		if err := rows.Scan(&msg.ID, &msg.Key, &msg.EventType, &msg.Payload, &msg.OccurredAt); err != nil {
			rows.Close()

			return 0, fmt.Errorf("relay outbox: scan: %w", err)
		}

		msgs = append(msgs, msg)
		ids = append(ids, msg.ID)
	}

	rows.Close()

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("relay outbox: iterate: %w", err)
	}

	if len(msgs) == 0 {
		return 0, nil
	}

	if err := publish(ctx, msgs); err != nil {
		return 0, fmt.Errorf("relay outbox: publish: %w", err)
	}

	if _, err := tx.Exec(ctx, MARK_OUTBOX_PUBLISHED_QUERY, ids); err != nil {
		return 0, fmt.Errorf("relay outbox: mark published: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("relay outbox: commit: %w", err)
	}

	return len(msgs), nil
}
