package kafka

import (
	"encoding/json"
	"time"
)

// Envelope — конверт, в котором доменные события приезжают от других сервисов.
// Структура повторяет ту, что публикуют auth-, customer-, kyc- и
// ledger-service.
//
// Разбирается только конверт: Payload остаётся сырым JSON, потому что его
// форма зависит от типа события, а знать все формы всех сервисов
// notification-service не обязан.
type Envelope struct {
	ID         string          `json:"event_id"`
	Type       string          `json:"type"`
	Source     string          `json:"source"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

// payload — поля, которые встречаются в событиях разных сервисов и нужны
// уведомлению. Отсутствующие останутся пустыми: json.Unmarshal не считает
// пропущенное поле ошибкой, и это ровно то поведение, которое тут нужно -
// один разбор на все типы событий.
type payload struct {
	CustomerID string `json:"customer_id"`
	Email      string `json:"email"`
	UserID     string `json:"user_id"`
}
