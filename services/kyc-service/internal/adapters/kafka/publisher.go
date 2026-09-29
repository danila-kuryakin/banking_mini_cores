package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

const EVENT_TYPE_HEADER = "event_type"

// Publisher пишет события в топик по умолчанию из конфига writer-а.
type Publisher struct {
	writer *kafka.Writer
}

func NewPublisher(writer *kafka.Writer) *Publisher {
	return &Publisher{writer: writer}
}

// Publish отправляет пачку синхронно: вернулся nil - брокер принял все сообщения.
func (p *Publisher) Publish(ctx context.Context, msgs []models.OutboxMessage) error {
	out := make([]kafka.Message, 0, len(msgs))

	for _, msg := range msgs {
		value, err := json.Marshal(models.Envelope{
			EventID:    msg.ID.String(),
			EventType:  msg.EventType,
			OccurredAt: msg.OccurredAt.UTC(),
			Payload:    msg.Payload,
		})
		if err != nil {
			return fmt.Errorf("marshal event %s: %w", msg.ID, err)
		}

		out = append(out, kafka.Message{
			Key:     []byte(msg.Key),
			Value:   value,
			Headers: []kafka.Header{{Key: EVENT_TYPE_HEADER, Value: []byte(msg.EventType)}},
		})
	}

	return p.writer.WriteMessages(ctx, out...)
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}
