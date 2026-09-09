package domain

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrMinIOClientCreate = errors.New("failed to create minio client")
	ErrMinIOUnavailable  = errors.New("minio is unavailable or bucket was not checked")
	ErrBucketCreate      = errors.New("failed to create bucket")
	ErrMinIOHealthCheck  = errors.New("minio health check failed")
)

var (
	ErrConfigValueRequired = errors.New("config value is required")
)

// Ошибки хранилища. Адаптер переводит в них коды S3, чтобы app-слой не разбирал
// minio.ErrorResponse сам.
var (
	ErrObjectNotFound = errors.New("object not found in storage")
)

// Внутренние ошибки: репозиторий их возвращает, gRPC-слой ловит через errors.Is
// и подменяет статусами ниже.
var (
	ErrMetadataNotFound  = errors.New("file metadata not found")
	ErrFileTypeConfirmed = errors.New("file of this type is already confirmed")
)

// Ошибки, уходящие наружу.
var (
	ErrUserIDMustBeUUID  = status.Error(codes.InvalidArgument, "user_id must be a uuid")
	ErrFileIDMustBeUUID  = status.Error(codes.InvalidArgument, "file_id must be a uuid")
	ErrFileTypeRequired  = status.Error(codes.InvalidArgument, "type must be specified")
	ErrFilenameIsTooLong = status.Error(codes.InvalidArgument, "filename is longer than 255 characters")
	ErrLimitIsNegative   = status.Error(codes.InvalidArgument, "limit must not be negative")
	ErrOffsetIsNegative  = status.Error(codes.InvalidArgument, "offset must not be negative")
)

var (
	ErrMetadataMissing        = status.Error(codes.NotFound, "file not found")
	ErrFileNotUploaded        = status.Error(codes.FailedPrecondition, "file was not uploaded to storage")
	ErrFileNotConfirmed       = status.Error(codes.FailedPrecondition, "file is not confirmed yet")
	ErrFileIsEmpty            = status.Error(codes.InvalidArgument, "uploaded file is empty")
	ErrFileIsTooLarge         = status.Error(codes.InvalidArgument, "uploaded file is larger than 10 MB")
	ErrContentTypeUnsupported = status.Error(codes.InvalidArgument, "only jpeg, png and pdf files are supported")
	ErrFileAlreadyExists      = status.Error(codes.AlreadyExists, "file of this type is already confirmed")
)

var (
	ErrInitUploadFailed    = status.Error(codes.Internal, "failed to init upload")
	ErrConfirmUploadFailed = status.Error(codes.Internal, "failed to confirm upload")
	ErrDownloadURLFailed   = status.Error(codes.Internal, "failed to build download url")
	ErrListFilesFailed     = status.Error(codes.Internal, "failed to list files")
	ErrRequiredCheckFailed = status.Error(codes.Internal, "failed to check required files")
)
