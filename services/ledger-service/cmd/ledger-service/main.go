package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	service "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/adapters/grpc_server"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/adapters/kafka"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/config"
	ledgerint "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/interceptors"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/ledger/v1"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "ledger-service: %v\n", err)
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

	fmt.Printf("config: %+v\n", cfg)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	dbPool, err := conn.NewConnectionDB(cfg.Postgres)
	if err != nil {
		log.Fatalf("Не удаётся подключиться к db: %v", err)
	}
	defer dbPool.Close()

	// Продюсер необязателен: без брокеров в конфиге вернётся nil, и сервис
	// поднимется без шины, а публикация событий станет no-op.
	producer, err := kafka.NewProducer(cfg.Kafka, logger)
	if err != nil {
		log.Fatalf("Не удаётся подключиться к kafka: %v", err)
	}
	defer func() {
		// Close дожидается отправки буфера, поэтому события, записанные
		// перед остановкой, не теряются.
		if err := producer.Close(); err != nil {
			logger.Error("kafka: продюсер закрыт с ошибкой", slog.Any("error", err))
		}
	}()

	return grpc_server.NewServer(
		cfg.Server.GetAddr(),
		logger,

		grpc_server.WithServices(
			func(r grpc.ServiceRegistrar) {
				ledgerv1.RegisterLedgerServiceServer(r, service.NewLedger(dbPool, producer, logger))
			},
		),

		// Базовую цепочку (recovery, request id, metrics, logging, timeout)
		// подключает сам grpc_server. Здесь - только интерцепторы
		// ledger-service; они встают ближе к хендлеру, уже под защитой
		// recovery и под логами.
		grpc_server.WithUnaryInterceptors(
			ledgerint.Validate(),
		),

		// Хендлеры ходят только в свою БД - 15 секунд по умолчанию тут
		// избыточны, столько ждать клиенту нечего.
		grpc_server.WithHandlerTimeout(5*time.Second),
	)
}
