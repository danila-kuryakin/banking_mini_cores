package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/migrator"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/grpc/client"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/server"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/app/worker"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/migrations"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := models.ValidateStatusTransitions(); err != nil {
		logger.Error("invalid application status transitions", "error", err)
		os.Exit(1)
	}

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

	clients, err := client.NewClients(cfg)
	if err != nil {
		logger.Error("failed to create grpc clients", "error", err)
		os.Exit(1)
	}
	defer clients.Close()

	repo := repository.NewRepository(dbPool)
	serv := service.NewService(repo, clients)

	// Воркер живёт до того же сигнала, что и сервер: grpc_server.NewServer
	// блокирует до SIGINT/SIGTERM, после чего этот контекст отменяется.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	autoChecks := worker.NewAutoChecks(repo, clients, logger, cfg.AutoChecks.Interval, cfg.AutoChecks.BatchSize)
	go autoChecks.Run(ctx)

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
