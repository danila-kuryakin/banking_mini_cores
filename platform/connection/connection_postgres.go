package connection

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Умолчания пула. Считать их надо не на сервис, а на весь стек: соединения
// делят все сервисы разом, а max_connections у Postgres по умолчанию 100.
// MinConns держится открытым постоянно, поэтому он маленький - иначе несколько
// сервисов исчерпают лимит, не обслужив ни одного запроса.
const (
	defaultMaxConns          = 10
	defaultMinConns          = 2
	defaultMaxConnLifetime   = 1 * time.Hour    // Время жизни подключения, для его обновления
	defaultMaxConnIdleTime   = 30 * time.Minute // Простаивающее соединение закрывается (например, ночью)
	defaultHealthCheckPeriod = 15 * time.Minute // Как часто проверять соединения
)

func NewConnectionDB(cfg config.DataBaseConfig) (*pgxpool.Pool, error) {
	dsn := cfg.GetDSN()
	// Парсим и разбираем структуру конфига
	poolcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("Не удалось прочесть конфиг дб: %w", err)
	}

	// Настройка пула: что задано в конфиге - берём оттуда, остальное по умолчанию.
	poolcfg.MaxConns = orDefaultInt32(cfg.MaxConns, defaultMaxConns)
	poolcfg.MinConns = orDefaultInt32(cfg.MinConns, defaultMinConns)
	poolcfg.MaxConnLifetime = orDefaultDuration(cfg.MaxConnLifetime, defaultMaxConnLifetime)
	poolcfg.MaxConnIdleTime = orDefaultDuration(cfg.MaxConnIdleTime, defaultMaxConnIdleTime)
	poolcfg.HealthCheckPeriod = orDefaultDuration(cfg.HealthCheckPeriod, defaultHealthCheckPeriod)

	// pgxpool при MinConns > MaxConns ведёт себя неочевидно, поэтому чиним сами
	// и говорим об этом вслух - опечатка в конфиге не должна тихо съедаться.
	if poolcfg.MinConns > poolcfg.MaxConns {
		log.Printf("db: min_conns (%d) больше max_conns (%d), опускаю min_conns до max_conns",
			poolcfg.MinConns, poolcfg.MaxConns)
		poolcfg.MinConns = poolcfg.MaxConns
	}

	// Создаем Pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolcfg)
	if err != nil {
		return nil, fmt.Errorf("Не удалось создать Pool: %w", err)
	}

	// Пингуем дб, проверить жива она или нет
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Связь с дб не установлена: %w", err)
	}

	log.Printf("Подключение к db успешно (пул: min %d, max %d)", poolcfg.MinConns, poolcfg.MaxConns)
	return pool, nil
}

func orDefaultInt32(v, def int32) int32 {
	if v <= 0 {
		return def
	}
	return v
}

func orDefaultDuration(v, def time.Duration) time.Duration {
	if v <= 0 {
		return def
	}
	return v
}
