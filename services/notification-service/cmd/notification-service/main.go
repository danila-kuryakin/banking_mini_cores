package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	notificationint "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/interceptors"
	service "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/adapters/grpc_server"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/adapters/kafka"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/adapters/memory"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/config"
	notificationv1 "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/pb/gen/notification/v1"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "notification-service: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Хранилище в памяти вместо БД: сервис читает чужие события и показывает
	// их через ListNotifications, источником правды он не является.
	repo := memory.NewRepository()

	// Консьюмер необязателен: без брокеров или без списка топиков вернётся
	// nil, и сервис поднимется с одним gRPC-API.
	consumer, err := kafka.NewConsumer(cfg.Kafka, repo.Notification, logger)
	if err != nil {
		log.Fatalf("Не удаётся подключиться к kafka: %v", err)
	}

	// Свой обработчик сигнала, помимо того, что заводит grpc_server: по
	// SIGTERM консьюмер и сервер должны останавливаться вместе, иначе процесс
	// повиснет - gRPC уже закрылся, а чтение из топика продолжается.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})

	go func() {
		defer close(done)

		if err := consumer.Run(ctx); err != nil {
			logger.Error("kafka: consumer stopped with error", slog.Any("error", err))
		}
	}()

	defer func() {
		// Порядок важен: Close прерывает висящий FetchMessage, и только после
		// этого горутина Run способна завершиться. Ждать её до Close значило
		// бы ждать вечно.
		if err := consumer.Close(); err != nil {
			logger.Error("kafka: consumer closed with error", slog.Any("error", err))
		}

		<-done
	}()

	return grpc_server.NewServer(
		cfg.Server.GetAddr(),
		logger,

		grpc_server.WithServices(
			func(r grpc.ServiceRegistrar) {
				notificationv1.RegisterNotificationServiceServer(r, service.NewNotification(repo, logger))
			},
		),

		// Базовую цепочку (recovery, request id, metrics, logging, timeout)
		// подключает сам grpc_server. Здесь - только интерцепторы
		// notification-service; они встают ближе к хендлеру, уже под защитой
		// recovery и под логами.
		grpc_server.WithUnaryInterceptors(
			notificationint.Validate(),
		),

		// Единственный хендлер читает список из памяти - ждать тут нечего.
		grpc_server.WithHandlerTimeout(5*time.Second),
	)
}
