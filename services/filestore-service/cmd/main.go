package main

import (
	"context"
	"log/slog"
	"os"

	conn "github.com/danila-kuryakin/banking_mini_cores/platform/connection"
	"github.com/danila-kuryakin/banking_mini_cores/platform/migrator"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/adapters/server"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/adapters/storage_s3"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/migrations"
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

	minioClient, err := storage_s3.NewMinIOClient(cfg.MinIOConfig)
	if err != nil {
		logger.Error("failed to connect to minio", "error", err)
		os.Exit(1)
	}

	go storage_s3.WatchHealth(context.Background(), minioClient, cfg.MinIOConfig, func(err error) {
		logger.Warn("minio health check failed", "error", err)
	})

	storage := storage_s3.NewStorage(minioClient, cfg.MinIOConfig.BucketName)

	repo := repository.NewRepository(dbPool)
	serv := service.NewService(repo, storage)

	srv, err := server.NewServer(cfg, serv, logger)
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	if err := srv.Run(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
