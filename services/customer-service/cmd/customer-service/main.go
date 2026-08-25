package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	service "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/adapters"
	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/config"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "customer-service: %v\n", err)
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
				customerv1.RegisterCustomerServiceServer(r, service.NewCustomer(dbPool, logger))
			},
		),

		// Хендлеры ходят только в свою БД - 15 секунд по умолчанию тут
		// избыточны, столько ждать клиенту нечего.
		grpc_server.WithHandlerTimeout(5*time.Second),
	)
}
