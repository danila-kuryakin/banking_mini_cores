package dto

import "time"

type Notification struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type" example:"ledger.transaction_completed"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type ListNotificationsQuery struct {
	Limit  int32 `form:"limit"`
	Offset int32 `form:"offset"`
}

type ListNotifications struct {
	Notifications []Notification `json:"notifications"`
}
