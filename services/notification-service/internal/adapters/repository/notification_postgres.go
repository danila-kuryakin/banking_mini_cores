package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

type NotificationRepo struct {
	db *pgxpool.Pool
}

func NewNotificationRepo(db *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{db: db}
}

// Save: event_id и уведомления в одной транзакции, поэтому повтор события не задвоит уведомления.
func (r *NotificationRepo) Save(ctx context.Context, envelope models.Envelope, notifications []models.Notification) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("save notifications: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, INSERT_PROCESSED_EVENT_QUERY, envelope.EventID, envelope.EventType)
	if err != nil {
		return false, fmt.Errorf("save notifications: mark event: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return false, nil
	}

	for _, n := range notifications {
		_, err := tx.Exec(ctx, INSERT_NOTIFICATION_QUERY, n.ID, n.UserID, n.EventID, n.EventType, n.Text, n.CreatedAt)
		if err != nil {
			return false, fmt.Errorf("save notifications: insert: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("save notifications: commit: %w", err)
	}

	return true, nil
}

func (r *NotificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, page models.Page) ([]*models.Notification, error) {
	rows, err := r.db.Query(ctx, LIST_NOTIFICATIONS_QUERY, userID, page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	notifications := make([]*models.Notification, 0, page.Limit)

	for rows.Next() {
		var (
			n         models.Notification
			createdAt time.Time
		)

		if err := rows.Scan(&n.ID, &n.UserID, &n.EventID, &n.EventType, &n.Text, &createdAt); err != nil {
			return nil, fmt.Errorf("scan notifications: %w", err)
		}

		n.CreatedAt = createdAt.UTC()
		notifications = append(notifications, &n)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}

	return notifications, nil
}
