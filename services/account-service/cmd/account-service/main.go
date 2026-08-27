package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	accountint "github.com/danila-kuryakin/banking_mini_cores/services/account-service/interceptors"
	service "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/adapters"
	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/config"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/account/v1"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "account-service: %v\n", err)
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
				accountv1.RegisterAccountServiceServer(r, service.NewAccount(dbPool, logger))
			},
		),

		// Базовую цепочку (recovery, request id, metrics, logging, timeout)
		// подключает сам grpc_server. Здесь - только интерцепторы
		// account-service; они встают ближе к хендлеру, уже под защитой
		// recovery и под логами.
		grpc_server.WithUnaryInterceptors(
			accountint.Validate(),
		),

		// Хендлеры ходят только в свою БД - 15 секунд по умолчанию тут
		// избыточны, столько ждать клиенту нечего.
		grpc_server.WithHandlerTimeout(5*time.Second),
	)
}
