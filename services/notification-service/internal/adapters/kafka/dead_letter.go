package kafka

import (
	"context"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

// DeadLetter пишет в DLQ-топик из конфига writer-а исходные ключ, тело и заголовки плюс причину.
type DeadLetter struct {
	writer *kafka.Writer
}

func NewDeadLetter(writer *kafka.Writer) *DeadLetter {
	return &DeadLetter{writer: writer}
}

func (d *DeadLetter) Send(ctx context.Context, msg models.Message, reason error) error {
	headers := make([]kafka.Header, 0, len(msg.Headers)+5)
	for _, h := range msg.Headers {
		headers = append(headers, kafka.Header{Key: h.Key, Value: h.Value})
	}

	headers = append(headers,
		kafka.Header{Key: domain.DLQ_HEADER_SOURCE_TOPIC, Value: []byte(msg.Topic)},
		kafka.Header{Key: domain.DLQ_HEADER_PARTITION, Value: []byte(strconv.Itoa(msg.Partition))},
		kafka.Header{Key: domain.DLQ_HEADER_OFFSET, Value: []byte(strconv.FormatInt(msg.Offset, 10))},
		kafka.Header{Key: domain.DLQ_HEADER_ERROR, Value: []byte(reason.Error())},
		kafka.Header{Key: domain.DLQ_HEADER_FAILED_AT, Value: []byte(time.Now().UTC().Format(time.RFC3339))},
	)

	return d.writer.WriteMessages(ctx, kafka.Message{
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: headers,
	})
}

func (d *DeadLetter) Close() error {
	return d.writer.Close()
}
