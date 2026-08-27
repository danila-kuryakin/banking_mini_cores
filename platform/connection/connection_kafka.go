package connection

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/segmentio/kafka-go"
)

// Умолчания клиента Kafka. Они консервативные: банковское событие потерять
// дороже, чем подождать лишние сто миллисекунд, поэтому по умолчанию запись
// синхронная и подтверждается всеми репликами из ISR.
const (
	defaultKafkaBatchSize      = 100
	defaultKafkaBatchTimeout   = 200 * time.Millisecond
	defaultKafkaWriteTimeout   = 10 * time.Second
	defaultKafkaMaxAttempts    = 5
	defaultKafkaMinBytes       = 1
	defaultKafkaMaxBytes       = 10 << 20 // 10 MiB
	defaultKafkaMaxWait        = 500 * time.Millisecond
	defaultKafkaDialTimeout    = 5 * time.Second
	defaultKafkaPartitionWatch = 5 * time.Second
	defaultKafkaStartOffset    = kafka.LastOffset
	defaultKafkaCommitInterval = 0 // синхронный коммит оффсета
)

// NewKafkaWriter создаёт продюсера и проверяет, что брокеры отвечают.
//
// Writer сам по себе ленивый: без проверки сервис поднялся бы с недоступной
// шиной и узнал бы об этом только на первом событии, в проде - ночью. Поэтому
// связь проверяется здесь, как и в NewConnectionDB.
//
// Topic в конфиге - это топик по умолчанию. Сообщение с непустым Topic
// отправится туда, куда указано в нём, поэтому одним писателем можно
// обслуживать несколько топиков.
func NewKafkaWriter(cfg config.KafkaConfig) (*kafka.Writer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if err := PingKafka(cfg); err != nil {
		return nil, err
	}

	brokers := cfg.BrokerList()

	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{}, // события одного ключа - в одну партицию, значит в порядке
		RequiredAcks: requiredAcks(cfg.RequiredAcks),
		BatchSize:    orDefaultInt(cfg.BatchSize, defaultKafkaBatchSize),
		BatchTimeout: orDefaultDuration(cfg.BatchTimeout, defaultKafkaBatchTimeout),
		WriteTimeout: orDefaultDuration(cfg.WriteTimeout, defaultKafkaWriteTimeout),
		MaxAttempts:  orDefaultInt(cfg.MaxAttempts, defaultKafkaMaxAttempts),
		Async:        cfg.Async,

		// Записать в несуществующий топик нельзя, поэтому создаём его на лету.
		// В dev это удобно; в проде топики заводит инфраструктура, и там у
		// брокера auto-create отключён - флаг просто ни на что не влияет.
		AllowAutoTopicCreation: true,
	}

	// ClientID виден брокеру: по нему в его логах понятно, чей это трафик.
	// Ставим транспорт только когда имя задано, чтобы в остальных случаях
	// пользоваться общим kafka.DefaultTransport.
	if cfg.ClientID != "" {
		w.Transport = &kafka.Transport{
			ClientID:    cfg.ClientID,
			DialTimeout: orDefaultDuration(cfg.DialTimeout, defaultKafkaDialTimeout),
		}
	}

	log.Printf("Подключение к kafka успешно (брокеры: %v, топик: %q)", brokers, cfg.Topic)
	return w, nil
}

// NewKafkaReader создаёт консьюмера группы cfg.GroupID на топик topic.
//
// Топик передаётся аргументом, а не берётся из конфига: сервис-подписчик обычно
// читает не тот топик, в который пишет сам, и топиков у него может быть
// несколько - по читателю на каждый.
func NewKafkaReader(cfg config.KafkaConfig, topic string) (*kafka.Reader, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if topic == "" {
		return nil, fmt.Errorf("kafka: не задан топик для чтения")
	}

	if cfg.GroupID == "" {
		return nil, fmt.Errorf("kafka: не задан group_id, читать топик %q некому", topic)
	}

	if err := PingKafka(cfg); err != nil {
		return nil, err
	}

	startOffset := defaultKafkaStartOffset
	if cfg.StartOffset == "first" {
		startOffset = kafka.FirstOffset
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.BrokerList(),
		Dialer: &kafka.Dialer{
			ClientID:  cfg.ClientID,
			Timeout:   orDefaultDuration(cfg.DialTimeout, defaultKafkaDialTimeout),
			DualStack: true,
		},
		GroupID:        cfg.GroupID,
		Topic:          topic,
		MinBytes:       orDefaultInt(cfg.MinBytes, defaultKafkaMinBytes),
		MaxBytes:       orDefaultInt(cfg.MaxBytes, defaultKafkaMaxBytes),
		MaxWait:        orDefaultDuration(cfg.MaxWait, defaultKafkaMaxWait),
		CommitInterval: orDefaultDuration(cfg.CommitInterval, defaultKafkaCommitInterval),

		// StartOffset учитывается только когда у группы ещё нет сохранённого
		// оффсета. После первого коммита группа продолжает с него.
		StartOffset: startOffset,

		// Ловит партиции, добавленные к уже существующему топику: с ростом
		// нагрузки их доливают, и читать новые кто-то должен.
		//
		// Случай "топика ещё не было в момент входа в группу" этим НЕ лечится -
		// проверено. Читатель остаётся без единой партиции и молча простаивает,
		// выглядя при этом живым. Поэтому топики заводятся заранее, задачей
		// kafka-init из services/kafka-service.
		WatchPartitionChanges:  true,
		PartitionWatchInterval: orDefaultDuration(cfg.PartitionWatchInterval, defaultKafkaPartitionWatch),
	})

	log.Printf("Подписка на kafka успешна (брокеры: %v, топик: %q, группа: %q)", cfg.BrokerList(), topic, cfg.GroupID)
	return r, nil
}

// PingKafka проверяет, что хотя бы один брокер из списка отвечает.
//
// Достаточно одного: адреса остальных клиент возьмёт из метаданных кластера,
// а недоступность конкретной ноды - это штатная ситуация, из-за которой сервис
// падать не должен.
func PingKafka(cfg config.KafkaConfig) error {
	timeout := orDefaultDuration(cfg.DialTimeout, defaultKafkaDialTimeout)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var lastErr error

	brokers := cfg.BrokerList()

	for _, broker := range brokers {
		conn, err := kafka.DialContext(ctx, "tcp", broker)
		if err != nil {
			lastErr = err
			continue
		}

		// Ошибку закрытия глотать нехорошо, но и делать с ней тут нечего:
		// соединение одноразовое, проверка уже прошла.
		_ = conn.Close()

		return nil
	}

	return fmt.Errorf("Связь с kafka не установлена (брокеры %v): %w", brokers, lastErr)
}

// requiredAcks переводит настройку конфига в константу клиента. Значения уже
// проверены в KafkaConfig.Validate, поэтому неизвестное здесь невозможно и
// пустая строка трактуется как самый безопасный вариант - ждать все реплики.
func requiredAcks(v string) kafka.RequiredAcks {
	switch v {
	case "none":
		return kafka.RequireNone
	case "leader":
		return kafka.RequireOne
	default:
		return kafka.RequireAll
	}
}

func orDefaultInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}
