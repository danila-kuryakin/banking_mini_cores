package models

import (
	"time"

	"github.com/google/uuid"
)

// Notification - уведомление клиенту; «отправка» - запись в таблицу и строка в логе.
type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	EventID   uuid.UUID
	EventType string
	Text      string
	CreatedAt time.Time
}

// Page - пагинация листинга: сколько записей взять и сколько пропустить.
type Page struct {
	Limit  int
	Offset int
}
