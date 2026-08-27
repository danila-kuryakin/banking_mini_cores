// Package kafka — исходящая шина customer-service: сюда складываются доменные
// события, которые сервис публикует наружу (смена профиля, смена статуса клиента). Знания о gRPC и о
// базе тут нет: адаптер умеет только упаковать событие в конверт и положить
// его в топик.
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/platform/connection"
)

// Envelope — общий конверт всех событий сервиса. Полезная нагрузка лежит
// внутри отдельным полем, чтобы подписчик мог разобрать конверт, посмотреть на
// Type и только потом решать, во что распаковывать Payload.
//
// ID нужен подписчику для дедупликации: Kafka гарантирует доставку
// "хотя бы один раз", то есть одно и то же событие может приехать дважды.
type Envelope struct {
	ID         string          `json:"event_id"`
	Type       string          `json:"type"`
	Source     string          `json:"source"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

// source проставляется в конверт: по нему в общем топике видно, кто автор.
const source = "customer-service"

// Producer публикует события customer-service в Kafka.
//
// Nil-указатель — рабочее состояние: если Kafka не настроена, main передаёт
// сюда nil, и все Publish превращаются в no-op. Так локальный запуск и тесты
// не требуют поднятого брокера, а вызывающему коду не приходится городить
// проверки вокруг каждой публикации.
type Producer struct {
	writer *kafkago.Writer
	log    *slog.Logger
}

// NewProducer подключается к брокерам и возвращает продюсера.
//
// Если брокеры в конфиге не заданы, возвращается (nil, nil): это не ошибка,
// а явно выбранный режим работы без шины.
func NewProducer(cfg config.KafkaConfig, log *slog.Logger) (*Producer, error) {
	if !cfg.Enabled() {
		log.Warn("kafka: no brokers configured, event publishing is disabled")
		return nil, nil
	}

	w, err := connection.NewKafkaWriter(cfg)
	if err != nil {
		return nil, fmt.Errorf("kafka: create producer: %w", err)
	}

	return &Producer{writer: w, log: log}, nil
}

// Close дожидается отправки того, что ещё лежит в буфере писателя.
func (p *Producer) Close() error {
	if p == nil {
		return nil
	}

	return p.writer.Close()
}

// publish упаковывает payload в конверт и отправляет его в топик по умолчанию.
//
// key задаёт партицию: события с одним ключом летят в одну партицию и потому
// приходят подписчику в том же порядке, в каком были опубликованы. Порядок
// между разными ключами Kafka не гарантирует - он и не нужен.
func (p *Producer) publish(ctx context.Context, eventType, key string, payload any) error {
	if p == nil {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("kafka: marshal payload for %s: %w", eventType, err)
	}

	env := Envelope{
		ID:         uuid.NewString(),
		Type:       eventType,
		Source:     source,
		OccurredAt: time.Now().UTC(),
		Payload:    body,
	}

	msg, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("kafka: marshal envelope for %s: %w", eventType, err)
	}

	err = p.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(key),
		Value: msg,
		Headers: []kafkago.Header{
			// Дублируем тип в заголовке: подписчик может отфильтровать
			// ненужное, не разбирая тело сообщения.
			{Key: "event_type", Value: []byte(eventType)},
			{Key: "event_id", Value: []byte(env.ID)},
		},
	})
	if err != nil {
		return fmt.Errorf("kafka: publish %s: %w", eventType, err)
	}

	p.log.Debug("kafka: event published",
		slog.String("type", eventType),
		slog.String("event_id", env.ID),
		slog.String("key", key),
	)

	return nil
}
