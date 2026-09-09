package interceptors

import (
	"context"
	"unicode/utf8"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/pb/gen/filestore/v1"
)

func Validate() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := validate(req); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func validate(req any) error {
	switch in := req.(type) {
	case *filestorev1.InitUploadRequest:
		if err := validateUserID(in.UserId); err != nil {
			return err
		}

		if in.Type == filestorev1.FileType_FILE_TYPE_UNSPECIFIED {
			return domain.ErrFileTypeRequired
		}

		return validateFilename(in.Filename)
	case *filestorev1.ConfirmUploadRequest:
		return validateIDs(in.UserId, in.FileId)
	case *filestorev1.GetDownloadUrlRequest:
		return validateIDs(in.UserId, in.FileId)
	case *filestorev1.ListFilesRequest:
		if err := validateUserID(in.UserId); err != nil {
			return err
		}

		return validatePage(in.Limit, in.Offset)
	case *filestorev1.HasRequiredFilesRequest:
		return validateUserID(in.UserId)
	default:
		return nil
	}
}

func validateIDs(userID, fileID string) error {
	if err := validateUserID(userID); err != nil {
		return err
	}

	if _, err := uuid.Parse(fileID); err != nil {
		return domain.ErrFileIDMustBeUUID
	}

	return nil
}

func validateUserID(userID string) error {
	if _, err := uuid.Parse(userID); err != nil {
		return domain.ErrUserIDMustBeUUID
	}

	return nil
}

func validateFilename(filename string) error {
	if utf8.RuneCountInString(filename) > domain.FILENAME_MAX_LENGTH {
		return domain.ErrFilenameIsTooLong
	}

	return nil
}

func validatePage(limit, offset int32) error {
	switch {
	case limit < 0:
		return domain.ErrLimitIsNegative
	case offset < 0:
		return domain.ErrOffsetIsNegative
	default:
		return nil
	}
}
