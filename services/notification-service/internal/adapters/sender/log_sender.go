package sender

import (
	"context"
	"log/slog"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

// LogSender - мок отправки: SMS и почты нет, уведомление просто пишется в лог.
type LogSender struct {
	log *slog.Logger
}

func NewLogSender(log *slog.Logger) *LogSender {
	return &LogSender{log: log}
}

func (s *LogSender) Send(_ context.Context, notification models.Notification) error {
	s.log.Info("notification sent",
		"notification_id", notification.ID,
		"user_id", notification.UserID,
		"event_type", notification.EventType,
		"text", notification.Text,
	)

	return nil
}
