package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Типы событий леджера в топике ledger.events.
const (
	EVENT_TRANSACTION_COMPLETED = "ledger.transaction_completed"
	EVENT_TRANSACTION_DECLINED  = "ledger.transaction_declined"
)

// OutboxMessage - событие, записанное в outbox в одной транзакции с проводками.
type OutboxMessage struct {
	ID         uuid.UUID
	Key        string
	EventType  string
	Payload    json.RawMessage
	OccurredAt time.Time
}

// Envelope - конверт события в Kafka, event_id нужен консьюмерам для дедупликации.
type Envelope struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

// TransactionEvent - payload событий transaction_completed и transaction_declined.
type TransactionEvent struct {
	TransactionID string    `json:"transaction_id"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	FromAccountID string    `json:"from_account_id"`
	ToAccountID   string    `json:"to_account_id"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	InitiatorID   string    `json:"initiator_id,omitempty"`
	DeclineReason string    `json:"decline_reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewTransactionEvent(tx *Transaction) (OutboxMessage, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return OutboxMessage{}, err
	}

	event := TransactionEvent{
		TransactionID: tx.ID.String(),
		Type:          string(tx.Type),
		Status:        string(tx.Status),
		FromAccountID: tx.FromAccountID.String(),
		ToAccountID:   tx.ToAccountID.String(),
		Amount:        tx.Amount,
		Currency:      string(tx.Currency),
		DeclineReason: tx.DeclineReason,
		CreatedAt:     tx.CreatedAt,
	}

	if tx.InitiatorID != nil {
		event.InitiatorID = tx.InitiatorID.String()
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return OutboxMessage{}, err
	}

	eventType := EVENT_TRANSACTION_COMPLETED
	if tx.Status == STATUS_DECLINED {
		eventType = EVENT_TRANSACTION_DECLINED
	}

	key := tx.FromAccountID
	if tx.Type == TYPE_DEPOSIT {
		key = tx.ToAccountID
	}

	return OutboxMessage{
		ID:         id,
		Key:        key.String(),
		EventType:  eventType,
		Payload:    payload,
		OccurredAt: tx.CreatedAt,
	}, nil
}
