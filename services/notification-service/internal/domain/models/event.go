package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Header - заголовок сообщения Kafka.
type Header struct {
	Key   string
	Value []byte
}

// Message - сообщение из Kafka; партиция и оффсет нужны для коммита и для DLQ.
type Message struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   []Header
}

// Envelope - общий конверт событий всех продюсеров.
type Envelope struct {
	EventID    uuid.UUID       `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
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
