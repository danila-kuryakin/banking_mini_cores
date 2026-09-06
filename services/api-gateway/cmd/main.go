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

	// Регистрирует сгенерированную спецификацию в реестре swag - без этого
	// /swagger/doc.json отдаёт 404. Файлы создаёт "make swagger".
	_ "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/docs"
)

//	@title						Banking Mini Cores API Gateway
//	@version					1.0
//	@description				HTTP-фасад над gRPC-сервисами: аутентификация и работа с клиентами.
//	@description				Все защищённые методы ждут заголовок Authorization: Bearer <access_token>.

//	@contact.name				Danila Kuryakin
//	@contact.url				https://github.com/danila-kuryakin/banking_mini_cores

//	@host						localhost:8080
//	@BasePath					/api/v1
//	@schemes					http

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Access-токен из POST /auth/login в формате "Bearer <token>".

//	@tag.name					auth
//	@tag.description			Регистрация, вход, обновление и отзыв токенов
//	@tag.name					customer
//	@tag.description			Карточка клиента, профиль и статус

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
