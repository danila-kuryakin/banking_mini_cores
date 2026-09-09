package main

import (
	"log/slog"
	"os"

	_ "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/docs"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/auth"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/grpc"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/server"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
	gwconfig "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/config"
)

//	@title						Banking Mini Cores API Gateway
//	@version					1.0
//	@description				HTTP-фасад над gRPC-сервисами: аутентификация, работа с клиентами и их файлами.

//	@contact.name				Danila Kuryakin
//	@contact.url				https://github.com/danila-kuryakin/banking_mini_cores

//	@BasePath	/api/v1

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Access-токен из POST /auth/login в формате "Bearer <token>".

//	@tag.name					auth
//	@tag.description			Регистрация, вход, обновление и отзыв токенов
//	@tag.name					customer
//	@tag.description			Карточка клиента, профиль и статус
//	@tag.name					files
//	@tag.description			Загрузка и выдача файлов клиента через presigned-ссылки

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := gwconfig.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	clients, err := grpc.NewGRPCClients(cfg)
	if err != nil {
		logger.Error("failed to create grpc clients", "error", err)
		os.Exit(1)
	}
	defer clients.Close()

	verifier := auth.NewVerifier(cfg.JWKSURL, cfg.JWKSRefreshInterval)

	svc := service.NewService(clients, verifier, cfg.RequestTimeout)

	handlers := handler.NewHandler(svc)

	if err := server.NewServer(cfg, handlers, logger).Run(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
