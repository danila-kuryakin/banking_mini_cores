package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/adapters/storage_s3"
	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
)

type FileStorageService struct {
	repo    *repository.Repository
	storage *storage_s3.Storage
}

func NewFileStorageService(repo *repository.Repository, storage *storage_s3.Storage) *FileStorageService {
	return &FileStorageService{
		repo:    repo,
		storage: storage,
	}
}

// InitUpload сохраняет метаданные и выдаёт ссылку куда загружать файл в хранилище.
func (s *FileStorageService) InitUpload(ctx context.Context, userID uuid.UUID, fileType models.FileType, filename string) (*models.Metadata, *models.PresignedURL, error) {
	fileID := uuid.New()

	metadata, err := s.repo.Metadata.CreateMetadata(ctx, models.Metadata{
		ID:         fileID,
		UserID:     userID,
		Type:       fileType,
		ObjectPath: fmt.Sprintf(domain.OBJECT_PATH_FORMAT, userID, fileType, fileID),
		Filename:   filename,
	})
	if err != nil {
		return nil, nil, err
	}

	url, err := s.storage.MinIO.PresignedPutURL(ctx, metadata.ObjectPath, domain.UPLOAD_URL_TTL)
	if err != nil {
		return nil, nil, err
	}

	return metadata, &models.PresignedURL{
		URL:       url,
		ExpiresAt: time.Now().UTC().Add(domain.UPLOAD_URL_TTL),
	}, nil
}

// ConfirmUpload проверяет, что файл загружен, и фиксирует метаданные.
func (s *FileStorageService) ConfirmUpload(ctx context.Context, userID, fileID uuid.UUID) (*models.Metadata, error) {
	metadata, err := s.repo.Metadata.GetMetadata(ctx, fileID, userID)
	if err != nil {
		return nil, err
	}

	if metadata.Status == models.FILE_STATUS_CONFIRMED {
		return metadata, nil
	}

	info, err := s.storage.MinIO.StatObject(ctx, metadata.ObjectPath)
	if err != nil {
		if errors.Is(err, domain.ErrObjectNotFound) {
			return nil, domain.ErrFileNotUploaded
		}

		return nil, err
	}

	switch {
	case info.Size == 0:
		_ = s.storage.MinIO.RemoveObject(ctx, metadata.ObjectPath)
		return nil, domain.ErrFileIsEmpty
	case info.Size > domain.MAX_FILE_SIZE:
		_ = s.storage.MinIO.RemoveObject(ctx, metadata.ObjectPath)
		return nil, domain.ErrFileIsTooLarge
	}

	contentType, checksum, err := s.inspectObject(ctx, metadata.ObjectPath)
	if err != nil {
		if errors.Is(err, domain.ErrObjectNotFound) {
			return nil, domain.ErrFileNotUploaded
		}

		return nil, err
	}

	if !models.IsAllowedContentType(contentType) {
		_ = s.storage.MinIO.RemoveObject(ctx, metadata.ObjectPath)

		return nil, domain.ErrContentTypeUnsupported
	}

	return s.repo.Metadata.ConfirmMetadata(ctx, fileID, userID, contentType, info.Size, checksum)
}

// GetDownloadURL отдаёт ссылку на скачивание.
func (s *FileStorageService) GetDownloadURL(ctx context.Context, userID, fileID uuid.UUID) (*models.PresignedURL, error) {
	metadata, err := s.repo.Metadata.GetMetadata(ctx, fileID, userID)
	if err != nil {
		return nil, err
	}

	if metadata.Status != models.FILE_STATUS_CONFIRMED {
		return nil, domain.ErrFileNotConfirmed
	}

	url, err := s.storage.MinIO.PresignedGetURL(ctx, metadata.ObjectPath, domain.DOWNLOAD_URL_TTL)
	if err != nil {
		return nil, err
	}

	return &models.PresignedURL{
		URL:       url,
		ExpiresAt: time.Now().UTC().Add(domain.DOWNLOAD_URL_TTL),
	}, nil
}

// ListFiles Список файлов из базы данных с пагинацией
func (s *FileStorageService) ListFiles(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Metadata, error) {
	if limit <= 0 {
		limit = domain.DEFAULT_PAGE_SIZE
	} else if limit > domain.MAX_PAGE_SIZE {
		limit = domain.MAX_PAGE_SIZE
	}

	if offset < 0 {
		offset = 0
	}

	return s.repo.Metadata.ListMetadata(ctx, userID, limit, offset)
}

// HasRequiredFiles отвечает kyc-service, можно ли принимать заявку.
func (s *FileStorageService) HasRequiredFiles(ctx context.Context, userID uuid.UUID) (bool, []models.FileType, error) {
	confirmed, err := s.repo.Metadata.ListConfirmedTypes(ctx, userID)
	if err != nil {
		return false, nil, err
	}

	present := make(map[models.FileType]struct{}, len(confirmed))
	for _, fileType := range confirmed {
		present[fileType] = struct{}{}
	}

	missing := make([]models.FileType, 0, len(models.REQUIRED_FILE_TYPES))
	for _, required := range models.REQUIRED_FILE_TYPES {
		if _, ok := present[required]; !ok {
			missing = append(missing, required)
		}
	}

	return len(missing) == 0, missing, nil
}

// inspectObject читает файл, считая sha256 и определяя тип по первым байтам.
func (s *FileStorageService) inspectObject(ctx context.Context, objectPath string) (string, string, error) {
	body, err := s.storage.MinIO.GetObject(ctx, objectPath)
	if err != nil {
		return "", "", err
	}
	defer body.Close()

	head := make([]byte, domain.CONTENT_SNIFF_LENGTH)

	read, err := io.ReadFull(body, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", "", fmt.Errorf("read object head %s: %w", objectPath, err)
	}
	head = head[:read]

	hasher := sha256.New()
	hasher.Write(head)

	if _, err := io.Copy(hasher, body); err != nil {
		return "", "", fmt.Errorf("read object %s: %w", objectPath, err)
	}

	return detectContentType(head), hex.EncodeToString(hasher.Sum(nil)), nil
}

// detectContentType срезает параметры вроде "; charset=utf-8" - на выходе
// лежат чистые media type.
func detectContentType(head []byte) string {
	mediaType, _, err := mime.ParseMediaType(http.DetectContentType(head))
	if err != nil {
		return ""
	}

	return mediaType
}
