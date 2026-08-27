package config

import (
	"fmt"
	"time"
)

type DataBaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"ssl_mode"`

	// Настройки пула. Нулевое значение означает "как по умолчанию" - значения
	// подставит connection.NewConnectionDB, поэтому в config.yml эти ключи
	// можно не писать вовсе.
	//
	// Важно помнить, что лимит у Postgres общий на всех: max_connections по
	// умолчанию 100, а пул заводится в каждом сервисе. Отсюда скромный
	// MinConns - иначе десяток сервисов выбирает лимит, ничего ещё не сделав.
	MaxConns          int32         `mapstructure:"max_conns"`
	MinConns          int32         `mapstructure:"min_conns"`
	MaxConnLifetime   time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `mapstructure:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `mapstructure:"health_check_period"`
}

func (dbg *DataBaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbg.Host, dbg.Port, dbg.User, dbg.Password, dbg.DBName, dbg.SSLMode)
}
