package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/migrator"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/adapters/antifraud"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/adapters/grpc/client"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/adapters/kafka"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/adapters/server"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/app/worker"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/migrations"
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

	cash, err := repo.Ledger.EnsureCashAccounts(context.Background(), cfg.CurrencySet.List())
	if err != nil {
		logger.Error("failed to ensure cash accounts", "error", err)
		os.Exit(1)
	}

	writer, err := conn.NewKafkaWriter(cfg.Kafka)
	if err != nil {
		logger.Error("failed to connect to kafka", "error", err)
		os.Exit(1)
	}

	publisher := kafka.NewPublisher(writer)
	defer func() { _ = publisher.Close() }()

	clients, err := client.NewClients(cfg)
	if err != nil {
		logger.Error("failed to create grpc clients", "error", err)
		os.Exit(1)
	}
	defer clients.Close()

	serv := service.NewService(repo, clients, antifraud.NewAllowAll(), cfg, cash)

	// Воркер живёт до того же сигнала, что и сервер: grpc_server.NewServer блокирует до SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	relay := worker.NewOutboxRelay(repo.Outbox, publisher, logger, cfg.Outbox.Interval, cfg.Outbox.BatchSize)
	go relay.Run(ctx)

	logger.Info("ledger settings", "currencies", cfg.CurrencySet.List(), "topic", cfg.Kafka.Topic)

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
