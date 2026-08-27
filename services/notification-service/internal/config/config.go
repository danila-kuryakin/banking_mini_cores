package config

import (
	"log"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
)

// Config is the notification-service configuration beyond the common base.
//
// Postgres тут нет намеренно: уведомления живут в памяти процесса, потому что
// сервис - потребитель событий, а не источник правды. Перезапуск теряет
// накопленное, и это осознанная цена за отсутствие ещё одной базы в стеке.
type Config struct {
	Server config.Server      `mapstructure:"server"`
	Kafka  config.KafkaConfig `mapstructure:"kafka"`
}

// Load reads the configuration from the environment.
func Load() (*Config, error) {

	cfg, err := config.Read[Config]("./configs")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	return cfg, nil
}
