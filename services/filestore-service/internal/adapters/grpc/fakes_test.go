package grpc

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
)

var _ interfaces.FileStorageService = (*fakeFileStorageService)(nil)

// аргументы вызова InitUpload
type initUploadCall struct {
	userID   uuid.UUID
	fileType models.FileType
	filename string
}

// аргументы вызовов с владельцем и файлом
type fileCall struct {
	userID uuid.UUID
	fileID uuid.UUID
}

// аргументы вызова ListFiles
type listFilesCall struct {
	userID uuid.UUID
	limit  int
	offset int
}

// фейк сервиса файлов: заданные ответы и запись вызовов
type fakeFileStorageService struct {
	mu sync.Mutex

	initUploadMeta  *models.Metadata
	initUploadURL   *models.PresignedURL
	initUploadErr   error
	initUploadCalls []initUploadCall

	confirmMeta  *models.Metadata
	confirmErr   error
	confirmCalls []fileCall

	downloadURL   *models.PresignedURL
	downloadErr   error
	downloadCalls []fileCall

	listResult []*models.Metadata
	listErr    error
	listCalls  []listFilesCall

	hasOK      bool
	hasMissing []models.FileType
	hasErr     error
	hasCalls   []uuid.UUID
}

func (f *fakeFileStorageService) InitUpload(_ context.Context, userID uuid.UUID, fileType models.FileType, filename string) (*models.Metadata, *models.PresignedURL, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.initUploadCalls = append(f.initUploadCalls, initUploadCall{userID: userID, fileType: fileType, filename: filename})
	if f.initUploadErr != nil {
		return nil, nil, f.initUploadErr
	}
	return f.initUploadMeta, f.initUploadURL, nil
}

func (f *fakeFileStorageService) ConfirmUpload(_ context.Context, userID, fileID uuid.UUID) (*models.Metadata, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.confirmCalls = append(f.confirmCalls, fileCall{userID: userID, fileID: fileID})
	if f.confirmErr != nil {
		return nil, f.confirmErr
	}
	return f.confirmMeta, nil
}

func (f *fakeFileStorageService) GetDownloadURL(_ context.Context, userID, fileID uuid.UUID) (*models.PresignedURL, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.downloadCalls = append(f.downloadCalls, fileCall{userID: userID, fileID: fileID})
	if f.downloadErr != nil {
		return nil, f.downloadErr
	}
	return f.downloadURL, nil
}

func (f *fakeFileStorageService) ListFiles(_ context.Context, userID uuid.UUID, limit, offset int) ([]*models.Metadata, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls = append(f.listCalls, listFilesCall{userID: userID, limit: limit, offset: offset})
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResult, nil
}

func (f *fakeFileStorageService) HasRequiredFiles(_ context.Context, userID uuid.UUID) (bool, []models.FileType, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hasCalls = append(f.hasCalls, userID)
	if f.hasErr != nil {
		return false, nil, f.hasErr
	}
	return f.hasOK, f.hasMissing, nil
}

// totalCalls - сколько раз фейк вызывали по всем методам
func (f *fakeFileStorageService) totalCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.initUploadCalls) + len(f.confirmCalls) + len(f.downloadCalls) + len(f.listCalls) + len(f.hasCalls)
}
