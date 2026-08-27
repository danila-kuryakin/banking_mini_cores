package kafka

import (
	"context"
	"time"
)

// Типы событий, которые публикует customer-service. Схема имени -
// <сервис>.<сущность>.<что произошло>, в прошедшем времени: событие сообщает
// о свершившемся факте, а не просит что-то сделать.
const (
	EventProfileUpdated = "customer.profile.updated"
	EventStatusChanged  = "customer.status.changed"
)

// ProfileUpdated — профиль клиента изменён.
//
// Само содержимое профиля в событие не кладётся: там персональные данные, а
// топик читают все подписанные сервисы и хранят до истечения retention. Кому
// нужны подробности - сходит в GetCustomer.
type ProfileUpdated struct {
	CustomerID string    `json:"customer_id"`
	UserID     string    `json:"user_id,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// StatusChanged — статус клиента изменён.
//
// Прежний статус едет вместе с новым: подписчику этого достаточно, чтобы
// понять, что произошло, не поднимая собственную историю.
type StatusChanged struct {
	CustomerID string    `json:"customer_id"`
	OldStatus  string    `json:"old_status"`
	NewStatus  string    `json:"new_status"`
	ChangedAt  time.Time `json:"changed_at"`
}

// PublishProfileUpdated публикует событие об изменении профиля.
//
// Ключом берётся CustomerID: всё, что произойдёт с этим клиентом дальше,
// попадёт в ту же партицию и приедет подписчику по порядку.
func (p *Producer) PublishProfileUpdated(ctx context.Context, ev ProfileUpdated) error {
	return p.publish(ctx, EventProfileUpdated, ev.CustomerID, ev)
}

// PublishStatusChanged публикует событие о смене статуса клиента.
func (p *Producer) PublishStatusChanged(ctx context.Context, ev StatusChanged) error {
	return p.publish(ctx, EventStatusChanged, ev.CustomerID, ev)
}
