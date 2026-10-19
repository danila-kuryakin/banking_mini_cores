package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	platformconfig "github.com/danila-kuryakin/banking_mini_cores/platform/config"
)

const (
	TEST_BROKER       = "kafka:9092"
	TEST_TOPIC_LEDGER = "ledger.events"
	TEST_TOPIC_KYC    = "kyc.events"
	TEST_GROUP_ID     = "notification"
	TEST_DLQ_TOPIC    = "notification.dlq"
)

// validConfig - минимальный валидный конфиг, Consumer и RequestTimeout нулевые
func validConfig() *Config {
	return &Config{
		Kafka: platformconfig.KafkaConfig{
			Brokers: []string{TEST_BROKER},
			Topics:  []string{TEST_TOPIC_LEDGER, TEST_TOPIC_KYC},
			GroupID: TEST_GROUP_ID,
			Topic:   TEST_DLQ_TOPIC,
		},
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(cfg *Config)
		wantErr bool
	}{
		{name: "valid minimal", modify: func(cfg *Config) {}},
		{name: "valid with consumer and timeout", modify: func(cfg *Config) {
			cfg.Consumer = Consumer{MaxAttempts: 5, MinBackoff: 100 * time.Millisecond, MaxBackoff: 10 * time.Second}
			cfg.RequestTimeout = 5 * time.Second
		}},
		{name: "required acks empty", modify: func(cfg *Config) { cfg.Kafka.RequiredAcks = "" }},
		{name: "required acks none", modify: func(cfg *Config) { cfg.Kafka.RequiredAcks = "none" }},
		{name: "required acks leader", modify: func(cfg *Config) { cfg.Kafka.RequiredAcks = "leader" }},
		{name: "required acks all", modify: func(cfg *Config) { cfg.Kafka.RequiredAcks = "all" }},
		{name: "start offset empty", modify: func(cfg *Config) { cfg.Kafka.StartOffset = "" }},
		{name: "start offset first", modify: func(cfg *Config) { cfg.Kafka.StartOffset = "first" }},
		{name: "start offset last", modify: func(cfg *Config) { cfg.Kafka.StartOffset = "last" }},
		{name: "broker with spaces", modify: func(cfg *Config) { cfg.Kafka.Brokers = []string{" kafka:9092 "} }},
		{name: "brokers with empty element", modify: func(cfg *Config) { cfg.Kafka.Brokers = []string{"", TEST_BROKER} }},
		{name: "topic with spaces", modify: func(cfg *Config) { cfg.Kafka.Topics = []string{" ledger.events "} }},
		{name: "topics with empty element", modify: func(cfg *Config) { cfg.Kafka.Topics = []string{"", TEST_TOPIC_LEDGER} }},
		{name: "brokers nil", modify: func(cfg *Config) { cfg.Kafka.Brokers = nil }, wantErr: true},
		{name: "brokers empty slice", modify: func(cfg *Config) { cfg.Kafka.Brokers = []string{} }, wantErr: true},
		{name: "brokers only blank", modify: func(cfg *Config) { cfg.Kafka.Brokers = []string{"", "  "} }, wantErr: true},
		{name: "required acks invalid", modify: func(cfg *Config) { cfg.Kafka.RequiredAcks = "some" }, wantErr: true},
		{name: "start offset invalid", modify: func(cfg *Config) { cfg.Kafka.StartOffset = "middle" }, wantErr: true},
		{name: "topics nil", modify: func(cfg *Config) { cfg.Kafka.Topics = nil }, wantErr: true},
		{name: "topics empty slice", modify: func(cfg *Config) { cfg.Kafka.Topics = []string{} }, wantErr: true},
		{name: "topics only blank", modify: func(cfg *Config) { cfg.Kafka.Topics = []string{"", "  "} }, wantErr: true},
		{name: "group id empty", modify: func(cfg *Config) { cfg.Kafka.GroupID = "" }, wantErr: true},
		{name: "dlq topic empty", modify: func(cfg *Config) { cfg.Kafka.Topic = "" }, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.modify(cfg)

			err := cfg.Validate()

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
