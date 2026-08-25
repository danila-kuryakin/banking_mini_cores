// Команда migrate накатывает и откатывает схему customer-service.
//
// Конфиг читается тем же способом, что и в сервисе (configs/config.yml + .env
// + переменные окружения), поэтому запускать нужно из services/customer-service:
//
//	go run ./cmd/migrate -cmd=up
//	go run ./cmd/migrate -cmd=steps -n=-1
//	go run ./cmd/migrate -cmd=version
//	go run ./cmd/migrate -cmd=force -v=2
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	platformconfig "github.com/danila-kuryakin/banking_mini_cores/platform/config"
	"github.com/danila-kuryakin/banking_mini_cores/platform/migrator"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/migrations"
)

// migrationsDir — корень embed.FS из пакета migrations.
const migrationsDir = "."

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "customer-service migrate: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		command = flag.String("cmd", "up", "up | down | steps | version | force")
		steps   = flag.Int("n", 0, "число шагов для -cmd=steps: >0 вверх, <0 вниз")
		version = flag.Int("v", -1, "версия для -cmd=force")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	switch *command {
	case "up":
		if err := migrator.Up(cfg.Postgres, migrations.FS, migrationsDir); err != nil {
			return err
		}
	case "down":
		if err := migrator.Down(cfg.Postgres, migrations.FS, migrationsDir); err != nil {
			return err
		}
	case "steps":
		if err := migrator.Steps(cfg.Postgres, migrations.FS, migrationsDir, *steps); err != nil {
			return err
		}
	case "force":
		if *version < 0 {
			return fmt.Errorf("для -cmd=force нужен неотрицательный -v")
		}
		if err := migrator.Force(cfg.Postgres, migrations.FS, migrationsDir, *version); err != nil {
			return err
		}
	case "version":
		// Версию только печатаем, ниже общий вывод не нужен.
		return printVersion(cfg.Postgres, logger)
	default:
		return fmt.Errorf("неизвестная команда %q: допустимы up, down, steps, version, force", *command)
	}

	return printVersion(cfg.Postgres, logger)
}

// printVersion сообщает, на какой версии схема оказалась.
func printVersion(db platformconfig.DataBaseConfig, logger *slog.Logger) error {
	version, dirty, applied, err := migrator.Version(db, migrations.FS, migrationsDir)
	if err != nil {
		return err
	}
	if !applied {
		logger.Info("схема пуста, миграции не накатаны")
		return nil
	}
	logger.Info("версия схемы", "version", version, "dirty", dirty)
	return nil
}
