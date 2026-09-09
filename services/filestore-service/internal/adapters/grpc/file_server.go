package grpc

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/pb/gen/filestore/v1"
)

type FileServer struct {
	filestorev1.UnimplementedFileServiceServer
	service *service.Service
	log     *slog.Logger
}

func NewFileServer(service *service.Service, log *slog.Logger) *FileServer {
	return &FileServer{
		service: service,
		log:     log,
	}
}

func (s *FileServer) InitUpload(ctx context.Context, in *filestorev1.InitUploadRequest) (*filestorev1.InitUploadResponse, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrUserIDMustBeUUID
	}

	fileType := typeFromProto(in.Type)
	if fileType == "" {
		return nil, domain.ErrFileTypeRequired
	}

	metadata, upload, err := s.service.Metadata.InitUpload(ctx, userID, fileType, in.Filename)
	if err != nil {
		if _, isStatus := status.FromError(err); isStatus {
			return nil, err
		}

		s.log.Error("failed to init upload", "user_id", in.UserId, "type", fileType, "error", err)

		return nil, domain.ErrInitUploadFailed
	}

	return &filestorev1.InitUploadResponse{
		FileId:    metadata.ID.String(),
		UploadUrl: upload.URL,
		ExpiresAt: timestamppb.New(upload.ExpiresAt),
	}, nil
}

func (s *FileServer) ConfirmUpload(ctx context.Context, in *filestorev1.ConfirmUploadRequest) (*filestorev1.FileMetadata, error) {
	userID, fileID, err := parseIDs(in.UserId, in.FileId)
	if err != nil {
		return nil, err
	}

	metadata, err := s.service.Metadata.ConfirmUpload(ctx, userID, fileID)
	if err != nil {
		if errors.Is(err, domain.ErrMetadataNotFound) {
			return nil, domain.ErrMetadataMissing
		}

		if errors.Is(err, domain.ErrFileTypeConfirmed) {
			return nil, domain.ErrFileAlreadyExists
		}

		if _, isStatus := status.FromError(err); isStatus {
			return nil, err
		}

		s.log.Error("failed to confirm upload", "user_id", in.UserId, "file_id", in.FileId, "error", err)

		return nil, domain.ErrConfirmUploadFailed
	}

	return metadataToProto(metadata), nil
}

func (s *FileServer) GetDownloadUrl(ctx context.Context, in *filestorev1.GetDownloadUrlRequest) (*filestorev1.GetDownloadUrlResponse, error) {
	userID, fileID, err := parseIDs(in.UserId, in.FileId)
	if err != nil {
		return nil, err
	}

	download, err := s.service.Metadata.GetDownloadURL(ctx, userID, fileID)
	if err != nil {
		if errors.Is(err, domain.ErrMetadataNotFound) {
			return nil, domain.ErrMetadataMissing
		}

		if _, isStatus := status.FromError(err); isStatus {
			return nil, err
		}

		s.log.Error("failed to build download url", "user_id", in.UserId, "file_id", in.FileId, "error", err)

		return nil, domain.ErrDownloadURLFailed
	}

	return &filestorev1.GetDownloadUrlResponse{
		Url:       download.URL,
		ExpiresAt: timestamppb.New(download.ExpiresAt),
	}, nil
}

func (s *FileServer) ListFiles(ctx context.Context, in *filestorev1.ListFilesRequest) (*filestorev1.ListFilesResponse, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrUserIDMustBeUUID
	}

	list, err := s.service.Metadata.ListFiles(ctx, userID, int(in.Limit), int(in.Offset))
	if err != nil {
		if _, isStatus := status.FromError(err); isStatus {
			return nil, err
		}

		s.log.Error("failed to list files", "user_id", in.UserId, "error", err)

		return nil, domain.ErrListFilesFailed
	}

	items := make([]*filestorev1.FileMetadata, 0, len(list))
	for _, metadata := range list {
		items = append(items, metadataToProto(metadata))
	}

	return &filestorev1.ListFilesResponse{
		Files: items,
	}, nil
}

func (s *FileServer) HasRequiredFiles(ctx context.Context, in *filestorev1.HasRequiredFilesRequest) (*filestorev1.HasRequiredFilesResponse, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrUserIDMustBeUUID
	}

	ok, missing, err := s.service.Metadata.HasRequiredFiles(ctx, userID)
	if err != nil {
		if _, isStatus := status.FromError(err); isStatus {
			return nil, err
		}

		s.log.Error("failed to check required files", "user_id", in.UserId, "error", err)

		return nil, domain.ErrRequiredCheckFailed
	}

	missingProto := make([]filestorev1.FileType, 0, len(missing))
	for _, fileType := range missing {
		missingProto = append(missingProto, typeToProto(fileType))
	}

	return &filestorev1.HasRequiredFilesResponse{
		Ok:      ok,
		Missing: missingProto,
	}, nil
}

func parseIDs(userID, fileID string) (uuid.UUID, uuid.UUID, error) {
	user, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, domain.ErrUserIDMustBeUUID
	}

	file, err := uuid.Parse(fileID)
	if err != nil {
		return uuid.Nil, uuid.Nil, domain.ErrFileIDMustBeUUID
	}

	return user, file, nil
}
