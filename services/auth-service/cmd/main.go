package main

import (
	"context"
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
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	dbPool, err := conn.NewConnectionDB(cfg.Postgres)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := migrator.Up(cfg.Postgres, migrations.FS, migrations.Dir); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	tokenManager, err := token.NewManager(cfg.JWT.PrivateKeyPath, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL, logger)
	if err != nil {
		logger.Error("failed to prepare jwt private key", "error", err)
		os.Exit(1)
	}

	repo := repository.NewRepository(dbPool)
	serv := service.NewService(repo, tokenManager)

	ensureAdmin(context.Background(), serv, cfg.Admin, logger)

	srv, err := server.NewServer(cfg, serv, tokenManager.JWKS(), logger)
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	if err := srv.Run(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// ensureAdmin создаёт администратора по умолчанию при первом запуске.
func ensureAdmin(ctx context.Context, serv *service.Service, admin config.Admin, logger *slog.Logger) {
	if admin.Password == "" {
		logger.Warn("admin password is not set, default admin is not created",
			"hint", "set ADMIN_PASSWORD to create it")

		return
	}

	created, err := serv.Auth.EnsureAdmin(ctx, admin.Email, admin.Password)
	if err != nil {
		logger.Error("failed to ensure default admin", "email", admin.Email, "error", err)

		return
	}

	if created {
		logger.Info("default admin created", "email", admin.Email)

		return
	}

	logger.Info("default admin already exists", "email", admin.Email)
}
