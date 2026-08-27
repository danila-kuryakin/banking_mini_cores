// Package notification — доменная модель сервиса: уведомление и его
// жизненный цикл. Про Kafka, gRPC и protobuf тут не знают ничего, чтобы
// адаптеры зависели от домена, а не домен от них.
package notification

import "time"

// Channel — канал доставки.
type Channel string

const (
	ChannelUnspecified Channel = ""
	ChannelEmail       Channel = "email"
	ChannelSMS         Channel = "sms"
	ChannelPush        Channel = "push"
)

// Status — состояние доставки.
type Status string

const (
	StatusPending Status = "pending"
	StatusSent    Status = "sent"
	StatusFailed  Status = "failed"
)

// Notification — уведомление, построенное из доменного события чужого сервиса.
//
// EventID и EventType сохраняются как есть: по первому уведомление
// дедуплицируется (Kafka доставляет "хотя бы один раз"), по второму видно,
// что именно его породило. Для отладочного ListNotifications это главное.
type Notification struct {
	ID         string
	CustomerID string
	EventID    string
	EventType  string
	Channel    Channel
	Status     Status
	Subject    string
	Body       string
	Attempts   int32
	CreatedAt  time.Time
	SentAt     time.Time
}

// Filter описывает выборку для ListNotifications. Нулевые поля означают
// "не фильтровать по этому признаку".
type Filter struct {
	CustomerID string
	Status     Status
}

// Matches сообщает, проходит ли уведомление фильтр.
func (f Filter) Matches(n Notification) bool {
	if f.CustomerID != "" && n.CustomerID != f.CustomerID {
		return false
	}

	if f.Status != "" && n.Status != f.Status {
		return false
	}

	return true
}
