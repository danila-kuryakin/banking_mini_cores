package service

import (
	"context"
	"time"

	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/filestore/v1"
)

type FilestoreService struct {
	filestoreClient filestorev1.FileServiceClient
	timeout         time.Duration
}

func NewFilestoreService(filestoreCli filestorev1.FileServiceClient, timeout time.Duration) *FilestoreService {
	return &FilestoreService{
		filestoreClient: filestoreCli,
		timeout:         timeout,
	}
}

func (s FilestoreService) InitUpload(ctx context.Context, userID string, fileType filestorev1.FileType, filename string) (*filestorev1.InitUploadResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.filestoreClient.InitUpload(ctx, &filestorev1.InitUploadRequest{
		UserId:   userID,
		Type:     fileType,
		Filename: filename,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s FilestoreService) ConfirmUpload(ctx context.Context, userID, fileID string) (*filestorev1.FileMetadata, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.filestoreClient.ConfirmUpload(ctx, &filestorev1.ConfirmUploadRequest{
		UserId: userID,
		FileId: fileID,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s FilestoreService) GetDownloadURL(ctx context.Context, userID, fileID string) (*filestorev1.GetDownloadUrlResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.filestoreClient.GetDownloadUrl(ctx, &filestorev1.GetDownloadUrlRequest{
		UserId: userID,
		FileId: fileID,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s FilestoreService) ListFiles(ctx context.Context, in *filestorev1.ListFilesRequest) ([]*filestorev1.FileMetadata, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	listed, err := s.filestoreClient.ListFiles(ctx, in)
	if err != nil {
		return nil, err
	}

	return listed.Files, nil
}

func (s FilestoreService) HasRequiredFiles(ctx context.Context, userID string) (*filestorev1.HasRequiredFilesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.filestoreClient.HasRequiredFiles(ctx, &filestorev1.HasRequiredFilesRequest{UserId: userID})
	if err != nil {
		return nil, err
	}

	return resp, nil
}
