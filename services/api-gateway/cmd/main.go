package main

import (
	"log/slog"
	"os"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/auth"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/grpc"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/server"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
	gwconfig "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/config"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := gwconfig.Load()
	if err != nil {
		logger.Error("Error loading config", "error", err)
		return
	}

	clients, err := grpc.NewGRPCClients(cfg)
	if err != nil {
		logger.Error("Clients error", "error", err)
		return
	}
	defer clients.Close()

	verifier := auth.NewVerifier(cfg.JWKSURL, cfg.JWKSRefreshInterval)

	svc := service.NewService(clients, verifier, cfg.RequestTimeout)

	handlers := handler.NewHandler(svc)

	err = server.NewServer(cfg, handlers, logger).Run()
	if err != nil {
		logger.Error("Server error", "error", err)
		return
	}
}
