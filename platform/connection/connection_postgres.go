package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewConnectionDB(cfg config.DataBaseConfig) (*pgxpool.Pool, error) {
	dsn := cfg.GetDSN()
	// Парсим и разбираем структуру конфига
	poolcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("Не удалось прочесть конфиг дб: %w", err)
	}

	// Настройка пула
	poolcfg.MaxConns = 100                       // Это макс.возможное подключение
	poolcfg.MinConns = 30                        // Это колчество соединений которое будет жить всегда
	poolcfg.MaxConnLifetime = 1 * time.Hour      // Время жизни подключения, для егоьобнолвения
	poolcfg.MaxConnIdleTime = 30 * time.Minute   // Если никто не юзает подключение оно вкл(например ночью)
	poolcfg.HealthCheckPeriod = 15 * time.Minute // Как часто будут проверки соединения

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

	log.Printf("Подключение к db успешно")
	return pool, nil
}
