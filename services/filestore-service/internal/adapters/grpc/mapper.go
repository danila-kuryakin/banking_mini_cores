package grpc

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/pb/gen/filestore/v1"
)

func metadataToProto(metadata *models.Metadata) *filestorev1.FileMetadata {
	return &filestorev1.FileMetadata{
		Id:          metadata.ID.String(),
		UserId:      metadata.UserID.String(),
		Type:        typeToProto(metadata.Type),
		Status:      statusToProto(metadata.Status),
		Filename:    metadata.Filename,
		ContentType: metadata.ContentType,
		SizeBytes:   metadata.SizeBytes,
		Sha256:      metadata.SHA256,
		CreatedAt:   timestamppb.New(metadata.CreatedAt),
		ConfirmedAt: timestampToProto(metadata.ConfirmedAt),
	}
}

func typeToProto(fileType models.FileType) filestorev1.FileType {
	switch fileType {
	case models.FILE_TYPE_PASSPORT:
		return filestorev1.FileType_FILE_TYPE_PASSPORT
	case models.FILE_TYPE_SELFIE:
		return filestorev1.FileType_FILE_TYPE_SELFIE
	case models.FILE_TYPE_PROOF_OF_ADDRESS:
		return filestorev1.FileType_FILE_TYPE_PROOF_OF_ADDRESS
	default:
		return filestorev1.FileType_FILE_TYPE_UNSPECIFIED
	}
}

func typeFromProto(fileType filestorev1.FileType) models.FileType {
	switch fileType {
	case filestorev1.FileType_FILE_TYPE_PASSPORT:
		return models.FILE_TYPE_PASSPORT
	case filestorev1.FileType_FILE_TYPE_SELFIE:
		return models.FILE_TYPE_SELFIE
	case filestorev1.FileType_FILE_TYPE_PROOF_OF_ADDRESS:
		return models.FILE_TYPE_PROOF_OF_ADDRESS
	default:
		return ""
	}
}

func statusToProto(status models.FileStatus) filestorev1.FileStatus {
	switch status {
	case models.FILE_STATUS_UPLOADED:
		return filestorev1.FileStatus_FILE_STATUS_UPLOADED
	case models.FILE_STATUS_CONFIRMED:
		return filestorev1.FileStatus_FILE_STATUS_CONFIRMED
	default:
		return filestorev1.FileStatus_FILE_STATUS_UNSPECIFIED
	}
}

// confirmed_at пустой, пока файл не подтверждён - в proto это отсутствующее
// поле, а не нулевой timestamp.
func timestampToProto(in *time.Time) *timestamppb.Timestamp {
	if in == nil {
		return nil
	}

	return timestamppb.New(*in)
}
