package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/migrator"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/adapters/kafka"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/adapters/server"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/app/worker"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/migrations"
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
	serv := service.NewService(repo, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for _, topic := range cfg.Kafka.TopicList() {
		kafkaReader, err := conn.NewKafkaReader(cfg.Kafka, topic)
		if err != nil {
			logger.Error("failed to subscribe to kafka", "topic", topic, "error", err)
			os.Exit(1)
		}

		reader := kafka.NewReader(kafkaReader)
		defer func() { _ = reader.Close() }()

		consumer := worker.NewDailyTotals(reader, repo.Totals, logger.With("topic", topic), cfg.Consumer.MinBackoff, cfg.Consumer.MaxBackoff)
		go consumer.Run(ctx)
	}

	logger.Info("antifraud settings", "currencies", cfg.CurrencySet.List(), "topics", cfg.Kafka.TopicList())

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
