package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
)

type MetadataRepo struct {
	db *pgxpool.Pool
}

func NewMetadataRepo(db *pgxpool.Pool) *MetadataRepo {
	return &MetadataRepo{db: db}
}

// CreateMetadata сохраняет метаданные перед загрузкой файла.
func (r *MetadataRepo) CreateMetadata(ctx context.Context, metadata models.Metadata) (*models.Metadata, error) {
	row := r.db.QueryRow(ctx, CREATE_METADATA_QUERY,
		metadata.ID,
		metadata.UserID,
		metadata.Type,
		metadata.ObjectPath,
		metadata.Filename,
	)

	created, err := scanMetadata(row)
	if err != nil {
		return nil, fmt.Errorf("create metadata: %w", err)
	}

	return created, nil
}

// GetMetadata ищет запись по паре file_id + user_id.
func (r *MetadataRepo) GetMetadata(ctx context.Context, fileID, userID uuid.UUID) (*models.Metadata, error) {
	row := r.db.QueryRow(ctx, GET_METADATA_QUERY, fileID, userID)

	metadata, err := scanMetadata(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMetadataNotFound
		}

		return nil, fmt.Errorf("get metadata: %w", err)
	}

	return metadata, nil
}

// ConfirmMetadata сохраняет метаданные после загрузки файла переводит и сохраняет статус confirmed.
func (r *MetadataRepo) ConfirmMetadata(ctx context.Context, fileID, userID uuid.UUID, contentType string, sizeBytes int64, sha256 string) (*models.Metadata, error) {
	row := r.db.QueryRow(ctx, CONFIRM_METADATA_QUERY,
		fileID,
		userID,
		contentType,
		sizeBytes,
		sha256,
	)

	metadata, err := scanMetadata(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, domain.ErrFileTypeConfirmed
		}

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMetadataNotFound
		}

		return nil, fmt.Errorf("confirm metadata: %w", err)
	}

	return metadata, nil
}

// ListMetadata отдаёт файлы пользователя.
func (r *MetadataRepo) ListMetadata(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Metadata, error) {
	rows, err := r.db.Query(ctx, LIST_METADATA_QUERY, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query metadata list: %w", err)
	}
	defer rows.Close()

	list := make([]*models.Metadata, 0, limit)

	for rows.Next() {
		metadata, err := scanMetadata(rows)
		if err != nil {
			return nil, fmt.Errorf("scan metadata list: %w", err)
		}

		list = append(list, metadata)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metadata list: %w", err)
	}

	return list, nil
}

// ListConfirmedTypes отдаёт типы, которые пользователь уже загрузил.
func (r *MetadataRepo) ListConfirmedTypes(ctx context.Context, userID uuid.UUID) ([]models.FileType, error) {
	rows, err := r.db.Query(ctx, LIST_CONFIRMED_TYPES_QUERY, userID)
	if err != nil {
		return nil, fmt.Errorf("query confirmed types: %w", err)
	}
	defer rows.Close()

	types := make([]models.FileType, 0, len(models.REQUIRED_FILE_TYPES))

	for rows.Next() {
		var fileType string
		if err := rows.Scan(&fileType); err != nil {
			return nil, fmt.Errorf("scan confirmed types: %w", err)
		}

		types = append(types, models.FileType(fileType))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate confirmed types: %w", err)
	}

	return types, nil
}

type scanner interface {
	Scan(dest ...any) error
}

// scanMetadata разбирает строку результата в модель, переводя type и status из текста в доменные типы.
func scanMetadata(row scanner) (*models.Metadata, error) {
	var (
		metadata   models.Metadata
		fileType   string
		fileStatus string
	)

	err := row.Scan(
		&metadata.ID,
		&metadata.UserID,
		&fileType,
		&fileStatus,
		&metadata.ObjectPath,
		&metadata.Filename,
		&metadata.ContentType,
		&metadata.SizeBytes,
		&metadata.SHA256,
		&metadata.CreatedAt,
		&metadata.ConfirmedAt,
	)
	if err != nil {
		return nil, err
	}

	metadata.Type = models.FileType(fileType)
	metadata.Status = models.FileStatus(fileStatus)

	return &metadata, nil
}
