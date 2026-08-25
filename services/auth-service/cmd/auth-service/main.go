package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/grpc_server"
	service "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/config"
	authv1 "github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/pb/gen/auth/v1"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "auth-service: %v\n", err)
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
		func(r grpc.ServiceRegistrar) {
			authv1.RegisterAuthServiceServer(r, service.NewAuth(dbPool, logger))
		},
	)
}
