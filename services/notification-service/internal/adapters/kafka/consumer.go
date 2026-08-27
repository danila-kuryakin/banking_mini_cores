// Package kafka — входящая шина notification-service.
//
// Сервис ничего не публикует: он слушает доменные события чужих сервисов и
// превращает их в уведомления. Отправки писем и SMS здесь пока нет, поэтому
// уведомление сохраняется со статусом pending - его видно через
// ListNotifications, и по нему понятно, что событие доехало.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/notification"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/repository"
)

// Consumer читает топики и складывает уведомления в хранилище.
//
// Читателей столько же, сколько топиков: у kafka-go один Reader обслуживает
// один топик. Все они в одной группе, поэтому при запуске второго инстанса
// сервиса партиции разделятся между ними, а не задвоятся.
type Consumer struct {
	readers []*kafkago.Reader
	repo    repository.Notification
	log     *slog.Logger
}

// NewConsumer подключается к брокерам и создаёт читателя на каждый топик.
//
// Если Kafka не настроена или список топиков пуст, возвращается (nil, nil):
// сервис поднимется с одним gRPC-API, и ListNotifications будет отдавать
// пустой список. Для локального запуска без брокера этого достаточно.
func NewConsumer(cfg config.KafkaConfig, repo repository.Notification, log *slog.Logger) (*Consumer, error) {
	topics := cfg.TopicList()

	if !cfg.Enabled() || len(topics) == 0 {
		log.Warn("kafka: no brokers or topics configured, event consumption is disabled")
		return nil, nil
	}

	readers := make([]*kafkago.Reader, 0, len(topics))

	for _, topic := range topics {
		r, err := connection.NewKafkaReader(cfg, topic)
		if err != nil {
			// Уже созданных читателей закрываем: иначе они останутся висеть
			// с открытыми соединениями, а сервис всё равно не поднимется.
			for _, opened := range readers {
				_ = opened.Close()
			}

			return nil, fmt.Errorf("kafka: create reader for topic %q: %w", topic, err)
		}

		readers = append(readers, r)
	}

	return &Consumer{readers: readers, repo: repo, log: log}, nil
}

// Run читает все топики, пока не отменят ctx. Возвращается только когда
// остановлены все читатели.
func (c *Consumer) Run(ctx context.Context) error {
	if c == nil {
		<-ctx.Done()
		return nil
	}

	g, ctx := errgroup.WithContext(ctx)

	for _, r := range c.readers {
		g.Go(func() error {
			return c.consume(ctx, r)
		})
	}

	return g.Wait()
}

// Close закрывает читателей. Вызывать после Run: Close прерывает висящий
// FetchMessage, и без него горутины остались бы ждать сообщения навсегда.
func (c *Consumer) Close() error {
	if c == nil {
		return nil
	}

	var errs []error

	for _, r := range c.readers {
		if err := r.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// consume — цикл по одному топику.
//
// FetchMessage, а не ReadMessage: оффсет коммитится вручную и только после
// того, как уведомление сохранено. Иначе падение между чтением и записью
// теряло бы событие - оффсет уже сдвинут, а уведомления нет.
func (c *Consumer) consume(ctx context.Context, r *kafkago.Reader) error {
	topic := r.Config().Topic

	c.log.Info("kafka: consuming topic", slog.String("topic", topic))

	for {
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			// Отмена контекста и закрытый читатель - это штатная остановка,
			// а не сбой: так сервис и должен завершаться по сигналу.
			if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
				c.log.Info("kafka: consumer stopped", slog.String("topic", topic))
				return nil
			}

			return fmt.Errorf("kafka: fetch from %q: %w", topic, err)
		}

		c.handle(ctx, topic, msg)

		if err := r.CommitMessages(ctx, msg); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
				return nil
			}

			return fmt.Errorf("kafka: commit offset for %q: %w", topic, err)
		}
	}
}

// handle разбирает сообщение и сохраняет уведомление.
//
// Ошибка разбора не возвращается наверх намеренно: битое сообщение уронило бы
// консьюмера, а после перезапуска он прочитал бы то же самое и упал снова.
// Такое сообщение логируется и пропускается; в проде его место - в
// dead-letter-топике.
func (c *Consumer) handle(ctx context.Context, topic string, msg kafkago.Message) {
	var env Envelope

	if err := json.Unmarshal(msg.Value, &env); err != nil {
		c.log.Error("kafka: malformed envelope, message skipped",
			slog.String("topic", topic),
			slog.Int64("offset", msg.Offset),
			slog.Any("error", err),
		)

		return
	}

	var p payload

	// Полезная нагрузка у каждого события своя; пустая или незнакомая - не
	// повод пропускать событие, уведомление всё равно будет создано.
	if len(env.Payload) > 0 {
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			c.log.Warn("kafka: malformed payload, notification will be incomplete",
				slog.String("topic", topic),
				slog.String("event_type", env.Type),
				slog.Any("error", err),
			)
		}
	}

	n := build(env, p)

	saved, err := c.repo.Save(ctx, n)
	if err != nil {
		c.log.Error("notification not saved",
			slog.String("event_id", env.ID),
			slog.Any("error", err),
		)

		return
	}

	if !saved {
		// Не ошибка: Kafka доставляет "хотя бы один раз", и повтор после
		// ребаланса группы - обычное дело.
		c.log.Debug("kafka: duplicate event skipped",
			slog.String("event_id", env.ID),
			slog.String("event_type", env.Type),
		)

		return
	}

	c.log.Info("notification created",
		slog.String("event_type", env.Type),
		slog.String("event_id", env.ID),
		slog.String("customer_id", n.CustomerID),
	)
}

// build собирает уведомление из события.
func build(env Envelope, p payload) notification.Notification {
	occurred := env.OccurredAt
	if occurred.IsZero() {
		occurred = time.Now().UTC()
	}

	// customer_id есть не во всех событиях: например, при регистрации клиента
	// ещё нет, и адресатом остаётся пользователь.
	customerID := p.CustomerID
	if customerID == "" {
		customerID = p.UserID
	}

	return notification.Notification{
		ID:         uuid.NewString(),
		CustomerID: customerID,
		EventID:    env.ID,
		EventType:  env.Type,
		Channel:    channelFor(env.Type),
		// Отправлять пока некому: адаптеров email/sms в сервисе нет, поэтому
		// уведомление остаётся в очереди на отправку.
		Status:    notification.StatusPending,
		Subject:   subjectFor(env.Type),
		Body:      string(env.Payload),
		CreatedAt: occurred,
	}
}
