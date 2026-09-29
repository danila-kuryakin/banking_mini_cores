package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/migrator"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/adapters/kafka"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/adapters/sender"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/adapters/server"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/worker"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/migrations"
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
	serv := service.NewService(repo, sender.NewLogSender(logger), cfg)

	writer, err := conn.NewKafkaWriter(cfg.Kafka)
	if err != nil {
		logger.Error("failed to connect to kafka", "error", err)
		os.Exit(1)
	}

	deadLetter := kafka.NewDeadLetter(writer)
	defer func() { _ = deadLetter.Close() }()

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

		consumer := worker.NewConsumer(
			reader,
			deadLetter,
			serv.Notifications.HandleEvent,
			logger.With("topic", topic),
			cfg.Consumer.MaxAttempts,
			cfg.Consumer.MinBackoff,
			cfg.Consumer.MaxBackoff,
		)
		go consumer.Run(ctx)
	}

	logger.Info("notification settings", "topics", cfg.Kafka.TopicList(), "dlq", cfg.Kafka.Topic)

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
