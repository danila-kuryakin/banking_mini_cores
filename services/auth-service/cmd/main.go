package main

import (
	"log/slog"
	"os"

	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/migrator"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/adapters/server"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/app/token"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/auth-service/migrations"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("error loading config", "error", err)
		return
	}

	dbPool, err := conn.NewConnectionDB(cfg.Postgres)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		return
	}
	defer dbPool.Close()

	if err := migrator.Up(cfg.Postgres, migrations.FS, migrations.Dir); err != nil {
		logger.Error("failed to run migrations", "error", err)
		return
	}

	tokenManager, err := token.NewManager(cfg.JWT.PrivateKeyPath, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL, logger)
	if err != nil {
		logger.Error("failed to prepare jwt private key", "error", err)
		return
	}

	repo := repository.NewRepository(dbPool)
	serv := service.NewService(repo, tokenManager)
	srv, err := server.NewServer(cfg, serv, tokenManager.JWKS(), logger)
	if err != nil {
		logger.Error("failed to create server", "error", err)
		return
	}

	if err := srv.Run(); err != nil {
		logger.Error("failed to run server", "error", err)
		return
	}
}
