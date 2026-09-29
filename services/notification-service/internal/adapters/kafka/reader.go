package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

// Reader - консьюмер группы на один топик; вендорный kafka.Message наружу не отдаётся.
type Reader struct {
	reader *kafka.Reader
}

func NewReader(reader *kafka.Reader) *Reader {
	return &Reader{reader: reader}
}

// Fetch не коммитит оффсет: сообщение считается прочитанным только после Commit.
func (r *Reader) Fetch(ctx context.Context) (models.Message, error) {
	msg, err := r.reader.FetchMessage(ctx)
	if err != nil {
		return models.Message{}, err
	}

	headers := make([]models.Header, 0, len(msg.Headers))
	for _, h := range msg.Headers {
		headers = append(headers, models.Header{Key: h.Key, Value: h.Value})
	}

	return models.Message{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   headers,
	}, nil
}

func (r *Reader) Commit(ctx context.Context, msg models.Message) error {
	return r.reader.CommitMessages(ctx, kafka.Message{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
	})
}

func (r *Reader) Close() error {
	return r.reader.Close()
}
