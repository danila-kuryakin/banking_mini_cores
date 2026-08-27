package config

import (
	"fmt"
	"strings"
	"time"
)

type KafkaConfig struct {
	// Brokers — список брокеров вида host:port.
	Brokers []string `mapstructure:"brokers"`

	// Topic — топик по умолчанию, в который пишет продюсер сервиса.
	Topic string `mapstructure:"topic"`

	// Topics — топики, которые сервис читает. Отдельно от Topic: писать сервис
	// обычно должен в один топик, а слушать - сразу несколько чужих.
	// Из окружения задаётся строкой через запятую:
	// KAFKA_TOPICS=auth.events,kyc.events.
	Topics []string `mapstructure:"topics"`

	// GroupID — группа консьюмера.
	GroupID string `mapstructure:"group_id"`

	// ClientID виден в логах и метриках брокера
	ClientID string `mapstructure:"client_id"`

	// --- Продюсер ---

	// RequiredAcks — чего ждать после записи: "none" (ничего), "leader"
	// (подтверждения лидера партиции) или "all" (всех реплик из ISR).
	// По умолчанию "all".
	RequiredAcks string `mapstructure:"required_acks"`

	// BatchSize и BatchTimeout — компромисс между задержкой и пропускной способностью.
	BatchSize    int           `mapstructure:"batch_size"`
	BatchTimeout time.Duration `mapstructure:"batch_timeout"`

	// WriteTimeout ограничивает одну попытку записи, MaxAttempts — сколько раз
	// повторять её при недоступности брокера или смене лидера партиции.
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	MaxAttempts  int           `mapstructure:"max_attempts"`

	// Async: true — Write возвращается сразу, ошибка доставки узнаётся только
	// из лога. Быстро, но событие можно потерять, поэтому по умолчанию false.
	Async bool `mapstructure:"async"`

	// --- Консьюмер ---

	// MinBytes и MaxBytes — сколько данных брокер отдаёт за один запрос.
	MinBytes int `mapstructure:"min_bytes"`
	MaxBytes int `mapstructure:"max_bytes"`

	// StartOffset — с чего начинать чтение новой группе: "first" (с начала
	// топика) или "last" (только новые сообщения).
	StartOffset string `mapstructure:"start_offset"`

	// MaxWait — сколько брокер держит запрос на чтение, ожидая, пока
	// накопится MinBytes.
	MaxWait time.Duration `mapstructure:"max_wait"`

	// CommitInterval: 0 — коммитить оффсет синхронно, по явному вызову.
	// Ненулевое значение включает фоновый коммит: быстрее, но после падения
	// часть уже обработанных сообщений приедет повторно.
	CommitInterval time.Duration `mapstructure:"commit_interval"`

	// PartitionWatchInterval — как часто перепроверять состав партиций топика.
	// Определяет, за сколько консьюмер заметит топик, созданный уже после его
	// подписки, и партиции, добавленные к существующему топику.
	PartitionWatchInterval time.Duration `mapstructure:"partition_watch_interval"`

	// DialTimeout ограничивает проверку связи с брокерами на старте сервиса.
	DialTimeout time.Duration `mapstructure:"dial_timeout"`
}

// BrokerList возвращает адреса брокеров, очищенные от пробелов и пустых
// элементов. Список приходит либо из YAML, либо строкой через запятую из
// окружения, и во втором случае "kafka:9092, kafka-2:9092" даёт элемент с
// ведущим пробелом - соединиться по такому адресу уже нельзя.
func (k KafkaConfig) BrokerList() []string {
	return trimList(k.Brokers)
}

// trimList чистит список, пришедший из YAML или из строки через запятую.
func trimList(in []string) []string {
	out := make([]string, 0, len(in))

	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}

	return out
}

// TopicList возвращает список читаемых топиков, очищенный от пробелов и
// пустых элементов - по той же причине, что и BrokerList.
func (k KafkaConfig) TopicList() []string {
	return trimList(k.Topics)
}

// Enabled сообщает, настроена ли Kafka вообще.
func (k KafkaConfig) Enabled() bool { return len(k.BrokerList()) > 0 }

// Validate проверяет то, что нельзя починить умолчанием.
func (k KafkaConfig) Validate() error {
	if !k.Enabled() {
		return fmt.Errorf("kafka: no brokers configured (brokers / KAFKA_BROKERS)")
	}

	switch k.RequiredAcks {
	case "", "none", "leader", "all":
	default:
		return fmt.Errorf("kafka: required_acks must be \"none\", \"leader\" or \"all\", got %q", k.RequiredAcks)
	}

	switch k.StartOffset {
	case "", "first", "last":
	default:
		return fmt.Errorf("kafka: start_offset must be \"first\" or \"last\", got %q", k.StartOffset)
	}

	return nil
}
