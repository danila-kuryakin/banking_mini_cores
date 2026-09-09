package storage_s3

import (
	"context"
	"fmt"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinIOClient(cfg config.MinIOConfig) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrMinIOClientCreate, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrMinIOUnavailable, err)
	}

	if !exists {
		err = client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("%w %s: %w", domain.ErrBucketCreate, cfg.BucketName, err)
		}
	}

	return client, nil
}

// WatchHealth раз в HealthCheckFreq проверяет связь с MinIO.
func WatchHealth(ctx context.Context, client *minio.Client, cfg config.MinIOConfig, onError func(error)) {
	ticker := time.NewTicker(cfg.HealthCheckFreq)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := check(ctx, client, cfg); err != nil {
				onError(err)
			}
		}
	}
}

// check выполняет одну проверку соединения: пробует прочитать первый объект бакета.
func check(ctx context.Context, client *minio.Client, cfg config.MinIOConfig) error {
	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	objectCh := client.ListObjects(ctx, cfg.BucketName, minio.ListObjectsOptions{
		MaxKeys: 1,
	})

	for object := range objectCh {
		if object.Err != nil {
			return fmt.Errorf("%w: %w", domain.ErrMinIOHealthCheck, object.Err)
		}
	}

	return nil
}
