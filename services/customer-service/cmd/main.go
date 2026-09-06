package main

import (
	"log/slog"
	"os"

	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/migrator"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/adapters/server"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/migrations"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	dbPool, err := conn.NewConnectionDB(cfg.Postgres)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := migrator.Up(cfg.Postgres, migrations.FS, migrations.Dir); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	repo := repository.NewRepository(dbPool)
	serv := service.NewService(repo)

	srv, err := server.NewServer(cfg, serv, logger)
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	if err := srv.Run(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
