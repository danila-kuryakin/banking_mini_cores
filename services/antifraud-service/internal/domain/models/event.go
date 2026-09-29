package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// События ledger-service, которые читает антифрод.
const (
	EVENT_TRANSACTION_COMPLETED = "ledger.transaction_completed"
	TRANSACTION_TYPE_TRANSFER   = "transfer"
)

// Message - сообщение из Kafka; партиция и оффсет нужны для коммита.
type Message struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte
}

// Envelope - конверт события, формат задаёт ledger-service.
type Envelope struct {
	EventID    uuid.UUID       `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

// TransactionPayload - payload событий ledger.transaction_*; антифроду нужна только его часть.
type TransactionPayload struct {
	TransactionID string    `json:"transaction_id"`
	Type          string    `json:"type"`
	FromAccountID uuid.UUID `json:"from_account_id"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	CreatedAt     time.Time `json:"created_at"`
}

// CompletedTransfer - проведённый перевод, который прибавляется к дневной сумме счёта.
type CompletedTransfer struct {
	EventID   uuid.UUID
	EventType string
	AccountID uuid.UUID
	Currency  Currency
	Amount    int64
	Day       time.Time
}

func ParseEnvelope(raw []byte) (Envelope, error) {
	var envelope Envelope

	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Envelope{}, fmt.Errorf("envelope: %w", err)
	}

	if envelope.EventID == uuid.Nil || envelope.EventType == "" {
		return Envelope{}, fmt.Errorf("envelope: event_id and event_type are required")
	}

	return envelope, nil
}

// Day - календарный день UTC, по нему копятся дневные суммы.
func Day(t time.Time) time.Time {
	return t.UTC().Truncate(24 * time.Hour)
}
