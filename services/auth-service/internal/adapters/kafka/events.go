package kafka

import (
	"context"
	"time"
)

// Типы событий, которые публикует auth-service. Схема имени -
// <сервис>.<сущность>.<что произошло>, в прошедшем времени: событие сообщает
// о свершившемся факте, а не просит что-то сделать.
const (
	EventUserRegistered = "auth.user.registered"
)

// UserRegistered — пользователь зарегистрирован.
//
// Пароля и токенов тут нет и быть не может: топик читают все желающие
// сервисы, а событие живёт в нём до истечения retention.
type UserRegistered struct {
	UserID     string    `json:"user_id"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	CustomerID string    `json:"customer_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// PublishUserRegistered публикует событие о регистрации.
//
// Ключом берётся UserID: всё, что произойдёт с этим пользователем дальше,
// попадёт в ту же партицию и приедет подписчику по порядку.
func (p *Producer) PublishUserRegistered(ctx context.Context, ev UserRegistered) error {
	return p.publish(ctx, EventUserRegistered, ev.UserID, ev)
}
