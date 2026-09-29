package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Типы событий заявки в топике kyc.events.
const (
	EVENT_APPLICATION_SUBMITTED = "kyc.application_submitted"
	EVENT_APPLICATION_APPROVED  = "kyc.application_approved"
	EVENT_APPLICATION_REJECTED  = "kyc.application_rejected"
	EVENT_DOCS_REQUESTED        = "kyc.docs_requested"
)

// eventTypes - статусы, о которых kyc сообщает наружу; авто-отказ и отказ офицера различает поле status.
var eventTypes = map[ApplicationStatus]string{
	STATUS_SUBMITTED:      EVENT_APPLICATION_SUBMITTED,
	STATUS_APPROVED:       EVENT_APPLICATION_APPROVED,
	STATUS_REJECTED:       EVENT_APPLICATION_REJECTED,
	STATUS_AUTO_REJECTED:  EVENT_APPLICATION_REJECTED,
	STATUS_DOCS_REQUESTED: EVENT_DOCS_REQUESTED,
}

// OutboxMessage - событие, записанное в outbox в одной транзакции со сменой статуса.
type OutboxMessage struct {
	ID         uuid.UUID
	Key        string
	EventType  string
	Payload    json.RawMessage
	OccurredAt time.Time
}

// Envelope - конверт события в Kafka, тот же, что у ledger-service.
type Envelope struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

// ApplicationEvent - payload событий заявки; пустой actor_id - системный переход.
type ApplicationEvent struct {
	ApplicationID  string    `json:"application_id"`
	UserID         string    `json:"user_id"`
	Status         string    `json:"status"`
	Reason         string    `json:"reason,omitempty"`
	RequestedTypes []string  `json:"requested_types,omitempty"`
	ActorID        string    `json:"actor_id,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NewApplicationEvent собирает событие о новом статусе; false - о таком статусе наружу не сообщаем.
func NewApplicationEvent(app *Application, actorID *uuid.UUID) (OutboxMessage, bool, error) {
	eventType, ok := eventTypes[app.Status]
	if !ok {
		return OutboxMessage{}, false, nil
	}

	id, err := uuid.NewV7()
	if err != nil {
		return OutboxMessage{}, false, err
	}

	event := ApplicationEvent{
		ApplicationID: app.ID.String(),
		UserID:        app.UserID.String(),
		Status:        string(app.Status),
		Reason:        app.Reason,
		UpdatedAt:     app.UpdatedAt.UTC(),
	}

	if app.Status == STATUS_DOCS_REQUESTED {
		for _, t := range app.RequestedTypes {
			event.RequestedTypes = append(event.RequestedTypes, string(t))
		}
	}

	if actorID != nil {
		event.ActorID = actorID.String()
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return OutboxMessage{}, false, err
	}

	// Ключ - клиент: события одной заявки и одного клиента идут в одну партицию по порядку.
	return OutboxMessage{
		ID:         id,
		Key:        app.UserID.String(),
		EventType:  eventType,
		Payload:    payload,
		OccurredAt: app.UpdatedAt.UTC(),
	}, true, nil
}
