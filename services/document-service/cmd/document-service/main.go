package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	service "github.com/danila-kuryakin/banking_mini_cores/services/document-service/internal/adapters"
	"github.com/danila-kuryakin/banking_mini_cores/services/document-service/internal/config"
	documentint "github.com/danila-kuryakin/banking_mini_cores/services/document-service/internal/interceptors"
	documentv1 "github.com/danila-kuryakin/banking_mini_cores/services/document-service/internal/pb/gen/document/v1"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "document-service: %v\n", err)
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

	return grpc_server.NewServer(
		cfg.Server.GetAddr(),
		logger,

		grpc_server.WithServices(
			func(r grpc.ServiceRegistrar) {
				documentv1.RegisterDocumentServiceServer(r, service.NewDocument(dbPool, logger))
			},
		),

		// Базовую цепочку (recovery, request id, metrics, logging, timeout)
		// подключает сам grpc_server. Здесь - только интерцепторы
		// document-service; они встают ближе к хендлеру, уже под защитой
		// recovery и под логами.
		grpc_server.WithUnaryInterceptors(
			documentint.Validate(),
		),

		// Хендлеры ходят только в свою БД - 15 секунд по умолчанию тут
		// избыточны, столько ждать клиенту нечего.
		grpc_server.WithHandlerTimeout(5*time.Second),
	)
}
