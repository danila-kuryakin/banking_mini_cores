package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/pb/gen/filestore/v1"
)

// тексты ошибок для проверки утечек и подстановки
const (
	TEST_DB_ERROR_TEXT = "db is down"
	TEST_PUT_URL       = "http://minio/put"
	TEST_GET_URL       = "http://minio/get"
)

// newTestFileServer собирает обработчик поверх фейка
func newTestFileServer(fake *fakeFileStorageService) *FileServer {
	return NewFileServer(&service.Service{Metadata: fake}, slog.New(slog.DiscardHandler))
}

// testExpiresAt - срок ссылки из примеров спецификации
func testExpiresAt() time.Time {
	return time.Date(2026, 1, 2, 3, 19, 5, 0, time.UTC)
}

// assertStatusErr проверяет sentinel, код и текст gRPC-статуса
func assertStatusErr(t *testing.T, err error, want error) {
	t.Helper()
	require.ErrorIs(t, err, want)
	assert.Equal(t, status.Code(want), status.Code(err))
	assert.Equal(t, status.Convert(want).Message(), status.Convert(err).Message())
	assert.NotContains(t, err.Error(), TEST_DB_ERROR_TEXT)
}

func TestFileServer_InitUpload(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	newFileID := uuid.MustParse(TEST_NEW_FILE_ID)
	dbErr := errors.New(TEST_DB_ERROR_TEXT)

	tests := []struct {
		name      string
		req       *filestorev1.InitUploadRequest
		svcErr    error
		want      *filestorev1.InitUploadResponse
		wantErr   error
		wantCalls []initUploadCall
	}{
		{
			name:      "proof of address with empty filename",
			req:       &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_PROOF_OF_ADDRESS, Filename: ""},
			want:      &filestorev1.InitUploadResponse{FileId: TEST_NEW_FILE_ID, UploadUrl: TEST_PUT_URL, ExpiresAt: timestamppb.New(testExpiresAt())},
			wantCalls: []initUploadCall{{userID: userID, fileType: TEST_TYPE_PROOF_OF_ADDRESS, filename: ""}},
		},
		{
			name:      "passport with cyrillic filename",
			req:       &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: "паспорт.pdf"},
			want:      &filestorev1.InitUploadResponse{FileId: TEST_NEW_FILE_ID, UploadUrl: TEST_PUT_URL, ExpiresAt: timestamppb.New(testExpiresAt())},
			wantCalls: []initUploadCall{{userID: userID, fileType: TEST_TYPE_PASSPORT, filename: "паспорт.pdf"}},
		},
		{
			name:      "selfie with spaces in filename",
			req:       &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_SELFIE, Filename: " a b.jpg "},
			want:      &filestorev1.InitUploadResponse{FileId: TEST_NEW_FILE_ID, UploadUrl: TEST_PUT_URL, ExpiresAt: timestamppb.New(testExpiresAt())},
			wantCalls: []initUploadCall{{userID: userID, fileType: TEST_TYPE_SELFIE, filename: " a b.jpg "}},
		},
		{
			name:    "unspecified type",
			req:     &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_UNSPECIFIED, Filename: "a"},
			wantErr: domain.ErrFileTypeRequired,
		},
		{
			name:    "type 4 out of enum",
			req:     &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType(4), Filename: "a"},
			wantErr: domain.ErrFileTypeUnknown,
		},
		{
			name:    "type 99 out of enum",
			req:     &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType(99), Filename: "a"},
			wantErr: domain.ErrFileTypeUnknown,
		},
		{
			name:    "negative type",
			req:     &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType(-1), Filename: "a"},
			wantErr: domain.ErrFileTypeUnknown,
		},
		{
			name:    "user_id not uuid",
			req:     &filestorev1.InitUploadRequest{UserId: TEST_BAD_ID, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: "a"},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name:    "user_id empty",
			req:     &filestorev1.InitUploadRequest{UserId: "", Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: "a"},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name:    "bad user_id checked before unknown type",
			req:     &filestorev1.InitUploadRequest{UserId: TEST_BAD_ID, Type: filestorev1.FileType(99), Filename: "a"},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name:    "bad user_id checked before unspecified type",
			req:     &filestorev1.InitUploadRequest{UserId: TEST_BAD_ID, Type: filestorev1.FileType_FILE_TYPE_UNSPECIFIED, Filename: "a"},
			wantErr: domain.ErrUserIDMustBeUUID,
		},
		{
			name:      "service status passed as is",
			req:       &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: "a"},
			svcErr:    domain.ErrContentTypeUnsupported,
			wantErr:   domain.ErrContentTypeUnsupported,
			wantCalls: []initUploadCall{{userID: userID, fileType: TEST_TYPE_PASSPORT, filename: "a"}},
		},
		{
			name:      "service plain error",
			req:       &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: "a"},
			svcErr:    dbErr,
			wantErr:   domain.ErrInitUploadFailed,
			wantCalls: []initUploadCall{{userID: userID, fileType: TEST_TYPE_PASSPORT, filename: "a"}},
		},
		{
			name:      "service wrapped error",
			req:       &filestorev1.InitUploadRequest{UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Filename: "a"},
			svcErr:    fmt.Errorf("x: %w", dbErr),
			wantErr:   domain.ErrInitUploadFailed,
			wantCalls: []initUploadCall{{userID: userID, fileType: TEST_TYPE_PASSPORT, filename: "a"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeFileStorageService{
				initUploadMeta: &models.Metadata{ID: newFileID, UserID: userID},
				initUploadURL:  &models.PresignedURL{URL: TEST_PUT_URL, ExpiresAt: testExpiresAt()},
				initUploadErr:  tt.svcErr,
			}
			resp, err := newTestFileServer(fake).InitUpload(t.Context(), tt.req)
			assert.Equal(t, tt.wantCalls, fake.initUploadCalls)
			if tt.wantErr != nil {
				assertStatusErr(t, err, tt.wantErr)
				assert.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.True(t, proto.Equal(tt.want, resp), "want %v, got %v", tt.want, resp)
			assert.True(t, resp.GetExpiresAt().AsTime().Equal(testExpiresAt()))
		})
	}
}

func TestFileServer_ConfirmUpload(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	fileID := uuid.MustParse(TEST_FILE_ID)
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	confirmedAt := time.Date(2026, 1, 2, 3, 5, 0, 0, time.UTC)
	dbErr := errors.New(TEST_DB_ERROR_TEXT)
	validReq := &filestorev1.ConfirmUploadRequest{UserId: TEST_USER_ID, FileId: TEST_FILE_ID}
	oneCall := []fileCall{{userID: userID, fileID: fileID}}

	confirmedMeta := &models.Metadata{
		ID: fileID, UserID: userID, Type: TEST_TYPE_SELFIE, Status: TEST_STATUS_CONFIRMED,
		ObjectPath: "u/selfie/f", Filename: "селфи.jpg", ContentType: "image/jpeg",
		SizeBytes: 2048, SHA256: "abc123", CreatedAt: createdAt, ConfirmedAt: &confirmedAt,
	}
	uploadedMeta := &models.Metadata{
		ID: fileID, UserID: userID, Type: TEST_TYPE_PASSPORT, Status: TEST_STATUS_UPLOADED, CreatedAt: createdAt,
	}

	tests := []struct {
		name      string
		req       *filestorev1.ConfirmUploadRequest
		svcMeta   *models.Metadata
		svcErr    error
		want      *filestorev1.FileMetadata
		wantErr   error
		wantCalls []fileCall
	}{
		{
			name:    "confirmed file",
			req:     validReq,
			svcMeta: confirmedMeta,
			want: &filestorev1.FileMetadata{
				Id: TEST_FILE_ID, UserId: TEST_USER_ID,
				Type: filestorev1.FileType_FILE_TYPE_SELFIE, Status: filestorev1.FileStatus_FILE_STATUS_CONFIRMED,
				ContentType: "image/jpeg", SizeBytes: 2048, Sha256: "abc123",
				CreatedAt: timestamppb.New(createdAt), ConfirmedAt: timestamppb.New(confirmedAt), Filename: "селфи.jpg",
			},
			wantCalls: oneCall,
		},
		{
			name:    "uploaded file without confirmed_at",
			req:     validReq,
			svcMeta: uploadedMeta,
			want: &filestorev1.FileMetadata{
				Id: TEST_FILE_ID, UserId: TEST_USER_ID,
				Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Status: filestorev1.FileStatus_FILE_STATUS_UPLOADED,
				CreatedAt: timestamppb.New(createdAt),
			},
			wantCalls: oneCall,
		},
		{name: "user_id not uuid", req: &filestorev1.ConfirmUploadRequest{UserId: TEST_BAD_ID, FileId: TEST_FILE_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "user_id empty", req: &filestorev1.ConfirmUploadRequest{UserId: "", FileId: TEST_FILE_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "file_id not uuid", req: &filestorev1.ConfirmUploadRequest{UserId: TEST_USER_ID, FileId: TEST_BAD_ID}, wantErr: domain.ErrFileIDMustBeUUID},
		{name: "file_id empty", req: &filestorev1.ConfirmUploadRequest{UserId: TEST_USER_ID, FileId: ""}, wantErr: domain.ErrFileIDMustBeUUID},
		{name: "both ids bad", req: &filestorev1.ConfirmUploadRequest{UserId: TEST_BAD_ID, FileId: TEST_BAD_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "metadata not found", req: validReq, svcErr: domain.ErrMetadataNotFound, wantErr: domain.ErrMetadataMissing, wantCalls: oneCall},
		{name: "metadata not found wrapped", req: validReq, svcErr: fmt.Errorf("confirm: %w", domain.ErrMetadataNotFound), wantErr: domain.ErrMetadataMissing, wantCalls: oneCall},
		{name: "type already confirmed", req: validReq, svcErr: domain.ErrFileTypeConfirmed, wantErr: domain.ErrFileAlreadyExists, wantCalls: oneCall},
		{name: "type already confirmed wrapped", req: validReq, svcErr: fmt.Errorf("confirm: %w", domain.ErrFileTypeConfirmed), wantErr: domain.ErrFileAlreadyExists, wantCalls: oneCall},
		{name: "status file not uploaded", req: validReq, svcErr: domain.ErrFileNotUploaded, wantErr: domain.ErrFileNotUploaded, wantCalls: oneCall},
		{name: "status file is empty", req: validReq, svcErr: domain.ErrFileIsEmpty, wantErr: domain.ErrFileIsEmpty, wantCalls: oneCall},
		{name: "status file is too large", req: validReq, svcErr: domain.ErrFileIsTooLarge, wantErr: domain.ErrFileIsTooLarge, wantCalls: oneCall},
		{name: "status content type unsupported", req: validReq, svcErr: domain.ErrContentTypeUnsupported, wantErr: domain.ErrContentTypeUnsupported, wantCalls: oneCall},
		{name: "plain error", req: validReq, svcErr: dbErr, wantErr: domain.ErrConfirmUploadFailed, wantCalls: oneCall},
		{name: "wrapped error", req: validReq, svcErr: fmt.Errorf("x: %w", dbErr), wantErr: domain.ErrConfirmUploadFailed, wantCalls: oneCall},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeFileStorageService{confirmMeta: tt.svcMeta, confirmErr: tt.svcErr}
			resp, err := newTestFileServer(fake).ConfirmUpload(t.Context(), tt.req)
			assert.Equal(t, tt.wantCalls, fake.confirmCalls)
			if tt.wantErr != nil {
				assertStatusErr(t, err, tt.wantErr)
				assert.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.True(t, proto.Equal(tt.want, resp), "want %v, got %v", tt.want, resp)
			assert.Equal(t, tt.svcMeta.ConfirmedAt == nil, resp.GetConfirmedAt() == nil)
		})
	}
}

func TestFileServer_GetDownloadUrl(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	fileID := uuid.MustParse(TEST_FILE_ID)
	dbErr := errors.New(TEST_DB_ERROR_TEXT)
	validReq := &filestorev1.GetDownloadUrlRequest{UserId: TEST_USER_ID, FileId: TEST_FILE_ID}
	oneCall := []fileCall{{userID: userID, fileID: fileID}}

	tests := []struct {
		name      string
		req       *filestorev1.GetDownloadUrlRequest
		svcErr    error
		want      *filestorev1.GetDownloadUrlResponse
		wantErr   error
		wantCalls []fileCall
	}{
		{
			name:      "download url",
			req:       validReq,
			want:      &filestorev1.GetDownloadUrlResponse{Url: TEST_GET_URL, ExpiresAt: timestamppb.New(testExpiresAt())},
			wantCalls: oneCall,
		},
		{name: "user_id not uuid", req: &filestorev1.GetDownloadUrlRequest{UserId: TEST_BAD_ID, FileId: TEST_FILE_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "user_id empty", req: &filestorev1.GetDownloadUrlRequest{UserId: "", FileId: TEST_FILE_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "file_id not uuid", req: &filestorev1.GetDownloadUrlRequest{UserId: TEST_USER_ID, FileId: TEST_BAD_ID}, wantErr: domain.ErrFileIDMustBeUUID},
		{name: "file_id empty", req: &filestorev1.GetDownloadUrlRequest{UserId: TEST_USER_ID, FileId: ""}, wantErr: domain.ErrFileIDMustBeUUID},
		{name: "both ids bad", req: &filestorev1.GetDownloadUrlRequest{UserId: TEST_BAD_ID, FileId: TEST_BAD_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "metadata not found", req: validReq, svcErr: domain.ErrMetadataNotFound, wantErr: domain.ErrMetadataMissing, wantCalls: oneCall},
		{name: "metadata not found wrapped", req: validReq, svcErr: fmt.Errorf("download: %w", domain.ErrMetadataNotFound), wantErr: domain.ErrMetadataMissing, wantCalls: oneCall},
		{name: "status file not confirmed", req: validReq, svcErr: domain.ErrFileNotConfirmed, wantErr: domain.ErrFileNotConfirmed, wantCalls: oneCall},
		{name: "plain error", req: validReq, svcErr: dbErr, wantErr: domain.ErrDownloadURLFailed, wantCalls: oneCall},
		{name: "wrapped error", req: validReq, svcErr: fmt.Errorf("x: %w", dbErr), wantErr: domain.ErrDownloadURLFailed, wantCalls: oneCall},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeFileStorageService{
				downloadURL: &models.PresignedURL{URL: TEST_GET_URL, ExpiresAt: testExpiresAt()},
				downloadErr: tt.svcErr,
			}
			resp, err := newTestFileServer(fake).GetDownloadUrl(t.Context(), tt.req)
			assert.Equal(t, tt.wantCalls, fake.downloadCalls)
			if tt.wantErr != nil {
				assertStatusErr(t, err, tt.wantErr)
				assert.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.True(t, proto.Equal(tt.want, resp), "want %v, got %v", tt.want, resp)
			assert.True(t, resp.GetExpiresAt().AsTime().Equal(testExpiresAt()))
		})
	}
}

func TestFileServer_ListFiles(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	fileID := uuid.MustParse(TEST_FILE_ID)
	newFileID := uuid.MustParse(TEST_NEW_FILE_ID)
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	confirmedAt := time.Date(2026, 1, 2, 3, 5, 0, 0, time.UTC)
	dbErr := errors.New(TEST_DB_ERROR_TEXT)

	twoFiles := []*models.Metadata{
		{ID: fileID, UserID: userID, Type: TEST_TYPE_PASSPORT, Status: TEST_STATUS_UPLOADED, ObjectPath: "u/passport/f", Filename: "p.pdf", CreatedAt: createdAt},
		{ID: newFileID, UserID: userID, Type: TEST_TYPE_SELFIE, Status: TEST_STATUS_CONFIRMED, ObjectPath: "u/selfie/f", Filename: "s.png", ContentType: "image/png", SizeBytes: 10, SHA256: "aa", CreatedAt: createdAt, ConfirmedAt: &confirmedAt},
	}
	twoFilesProto := []*filestorev1.FileMetadata{
		{Id: TEST_FILE_ID, UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_PASSPORT, Status: filestorev1.FileStatus_FILE_STATUS_UPLOADED, Filename: "p.pdf", CreatedAt: timestamppb.New(createdAt)},
		{Id: TEST_NEW_FILE_ID, UserId: TEST_USER_ID, Type: filestorev1.FileType_FILE_TYPE_SELFIE, Status: filestorev1.FileStatus_FILE_STATUS_CONFIRMED, Filename: "s.png", ContentType: "image/png", SizeBytes: 10, Sha256: "aa", CreatedAt: timestamppb.New(createdAt), ConfirmedAt: timestamppb.New(confirmedAt)},
	}

	tests := []struct {
		name      string
		req       *filestorev1.ListFilesRequest
		svcFiles  []*models.Metadata
		svcErr    error
		want      []*filestorev1.FileMetadata
		wantErr   error
		wantCalls []listFilesCall
	}{
		{
			name:      "two files in order",
			req:       &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: 20, Offset: 100},
			svcFiles:  twoFiles,
			want:      twoFilesProto,
			wantCalls: []listFilesCall{{userID: userID, limit: 20, offset: 100}},
		},
		{
			name:      "zero limit and offset",
			req:       &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: 0, Offset: 0},
			svcFiles:  []*models.Metadata{},
			wantCalls: []listFilesCall{{userID: userID, limit: 0, offset: 0}},
		},
		{
			name:      "max int32 limit",
			req:       &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: math.MaxInt32, Offset: 5},
			svcFiles:  []*models.Metadata{},
			wantCalls: []listFilesCall{{userID: userID, limit: 2147483647, offset: 5}},
		},
		{
			name:      "negative limit and offset passed as is",
			req:       &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: -1, Offset: -5},
			svcFiles:  []*models.Metadata{},
			wantCalls: []listFilesCall{{userID: userID, limit: -1, offset: -5}},
		},
		{
			name:      "nil slice from service",
			req:       &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: 20},
			svcFiles:  nil,
			wantCalls: []listFilesCall{{userID: userID, limit: 20, offset: 0}},
		},
		{name: "user_id not uuid", req: &filestorev1.ListFilesRequest{UserId: TEST_BAD_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "user_id empty", req: &filestorev1.ListFilesRequest{UserId: ""}, wantErr: domain.ErrUserIDMustBeUUID},
		{
			name:      "service status passed as is",
			req:       &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: 20},
			svcErr:    domain.ErrFileNotConfirmed,
			wantErr:   domain.ErrFileNotConfirmed,
			wantCalls: []listFilesCall{{userID: userID, limit: 20, offset: 0}},
		},
		{
			name:      "plain error",
			req:       &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: 20},
			svcErr:    dbErr,
			wantErr:   domain.ErrListFilesFailed,
			wantCalls: []listFilesCall{{userID: userID, limit: 20, offset: 0}},
		},
		{
			name:      "wrapped error",
			req:       &filestorev1.ListFilesRequest{UserId: TEST_USER_ID, Limit: 20},
			svcErr:    fmt.Errorf("x: %w", dbErr),
			wantErr:   domain.ErrListFilesFailed,
			wantCalls: []listFilesCall{{userID: userID, limit: 20, offset: 0}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeFileStorageService{listResult: tt.svcFiles, listErr: tt.svcErr}
			resp, err := newTestFileServer(fake).ListFiles(t.Context(), tt.req)
			assert.Equal(t, tt.wantCalls, fake.listCalls)
			if tt.wantErr != nil {
				assertStatusErr(t, err, tt.wantErr)
				assert.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp.GetFiles(), len(tt.want))
			for i := range tt.want {
				assert.True(t, proto.Equal(tt.want[i], resp.GetFiles()[i]), "index %d: want %v, got %v", i, tt.want[i], resp.GetFiles()[i])
			}
		})
	}
}

func TestFileServer_HasRequiredFiles(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	dbErr := errors.New(TEST_DB_ERROR_TEXT)
	validReq := &filestorev1.HasRequiredFilesRequest{UserId: TEST_USER_ID}
	oneCall := []uuid.UUID{userID}

	tests := []struct {
		name        string
		req         *filestorev1.HasRequiredFilesRequest
		svcOK       bool
		svcMissing  []models.FileType
		svcErr      error
		wantOK      bool
		wantMissing []filestorev1.FileType
		wantErr     error
		wantCalls   []uuid.UUID
	}{
		{
			name:        "passport and selfie missing",
			req:         validReq,
			svcOK:       false,
			svcMissing:  []models.FileType{TEST_TYPE_PASSPORT, TEST_TYPE_SELFIE},
			wantOK:      false,
			wantMissing: []filestorev1.FileType{filestorev1.FileType_FILE_TYPE_PASSPORT, filestorev1.FileType_FILE_TYPE_SELFIE},
			wantCalls:   oneCall,
		},
		{
			name:        "order kept",
			req:         validReq,
			svcOK:       false,
			svcMissing:  []models.FileType{TEST_TYPE_SELFIE, TEST_TYPE_PASSPORT},
			wantOK:      false,
			wantMissing: []filestorev1.FileType{filestorev1.FileType_FILE_TYPE_SELFIE, filestorev1.FileType_FILE_TYPE_PASSPORT},
			wantCalls:   oneCall,
		},
		{
			name:       "all present with empty slice",
			req:        validReq,
			svcOK:      true,
			svcMissing: []models.FileType{},
			wantOK:     true,
			wantCalls:  oneCall,
		},
		{
			name:       "all present with nil slice",
			req:        validReq,
			svcOK:      true,
			svcMissing: nil,
			wantOK:     true,
			wantCalls:  oneCall,
		},
		{name: "user_id not uuid", req: &filestorev1.HasRequiredFilesRequest{UserId: TEST_BAD_ID}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "user_id empty", req: &filestorev1.HasRequiredFilesRequest{UserId: ""}, wantErr: domain.ErrUserIDMustBeUUID},
		{name: "service status passed as is", req: validReq, svcErr: domain.ErrFileIsEmpty, wantErr: domain.ErrFileIsEmpty, wantCalls: oneCall},
		{name: "plain error", req: validReq, svcErr: dbErr, wantErr: domain.ErrRequiredCheckFailed, wantCalls: oneCall},
		{name: "wrapped error", req: validReq, svcErr: fmt.Errorf("x: %w", dbErr), wantErr: domain.ErrRequiredCheckFailed, wantCalls: oneCall},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeFileStorageService{hasOK: tt.svcOK, hasMissing: tt.svcMissing, hasErr: tt.svcErr}
			resp, err := newTestFileServer(fake).HasRequiredFiles(t.Context(), tt.req)
			assert.Equal(t, tt.wantCalls, fake.hasCalls)
			if tt.wantErr != nil {
				assertStatusErr(t, err, tt.wantErr)
				assert.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.wantOK, resp.GetOk())
			// nil и пустой срез здесь не различаются - сравниваем как непустые срезы
			assert.Equal(t, append([]filestorev1.FileType{}, tt.wantMissing...), append([]filestorev1.FileType{}, resp.GetMissing()...))
		})
	}
}

// serverCall вызывает один метод обработчика и отдаёт только ошибку
type serverCall func(ctx context.Context, s *FileServer) error

func callInitUpload(userID string, fileType filestorev1.FileType) serverCall {
	return func(ctx context.Context, s *FileServer) error {
		_, err := s.InitUpload(ctx, &filestorev1.InitUploadRequest{UserId: userID, Type: fileType, Filename: "a"})
		return err
	}
}

func callConfirmUpload(userID, fileID string) serverCall {
	return func(ctx context.Context, s *FileServer) error {
		_, err := s.ConfirmUpload(ctx, &filestorev1.ConfirmUploadRequest{UserId: userID, FileId: fileID})
		return err
	}
}

func callGetDownloadUrl(userID, fileID string) serverCall {
	return func(ctx context.Context, s *FileServer) error {
		_, err := s.GetDownloadUrl(ctx, &filestorev1.GetDownloadUrlRequest{UserId: userID, FileId: fileID})
		return err
	}
}

func callListFiles(userID string) serverCall {
	return func(ctx context.Context, s *FileServer) error {
		_, err := s.ListFiles(ctx, &filestorev1.ListFilesRequest{UserId: userID, Limit: 20})
		return err
	}
}

func callHasRequiredFiles(userID string) serverCall {
	return func(ctx context.Context, s *FileServer) error {
		_, err := s.HasRequiredFiles(ctx, &filestorev1.HasRequiredFilesRequest{UserId: userID})
		return err
	}
}

// failingFake - фейк, у которого каждый метод возвращает одну и ту же ошибку
func failingFake(err error) *fakeFileStorageService {
	return &fakeFileStorageService{initUploadErr: err, confirmErr: err, downloadErr: err, listErr: err, hasErr: err}
}

func TestFileServer_InvariantErrorIsAllowedStatus(t *testing.T) {
	allowed := []codes.Code{codes.InvalidArgument, codes.NotFound, codes.FailedPrecondition, codes.AlreadyExists, codes.Internal}
	dbErr := errors.New(TEST_DB_ERROR_TEXT)
	tests := []struct {
		name   string
		svcErr error
		call   serverCall
	}{
		{name: "init bad user", call: callInitUpload(TEST_BAD_ID, filestorev1.FileType_FILE_TYPE_PASSPORT)},
		{name: "init unspecified type", call: callInitUpload(TEST_USER_ID, filestorev1.FileType_FILE_TYPE_UNSPECIFIED)},
		{name: "init unknown type", call: callInitUpload(TEST_USER_ID, filestorev1.FileType(99))},
		{name: "init service status", svcErr: domain.ErrContentTypeUnsupported, call: callInitUpload(TEST_USER_ID, filestorev1.FileType_FILE_TYPE_PASSPORT)},
		{name: "init db error", svcErr: dbErr, call: callInitUpload(TEST_USER_ID, filestorev1.FileType_FILE_TYPE_PASSPORT)},
		{name: "confirm bad user", call: callConfirmUpload(TEST_BAD_ID, TEST_FILE_ID)},
		{name: "confirm bad file", call: callConfirmUpload(TEST_USER_ID, TEST_BAD_ID)},
		{name: "confirm not found", svcErr: domain.ErrMetadataNotFound, call: callConfirmUpload(TEST_USER_ID, TEST_FILE_ID)},
		{name: "confirm type confirmed", svcErr: domain.ErrFileTypeConfirmed, call: callConfirmUpload(TEST_USER_ID, TEST_FILE_ID)},
		{name: "confirm not uploaded", svcErr: domain.ErrFileNotUploaded, call: callConfirmUpload(TEST_USER_ID, TEST_FILE_ID)},
		{name: "confirm too large", svcErr: domain.ErrFileIsTooLarge, call: callConfirmUpload(TEST_USER_ID, TEST_FILE_ID)},
		{name: "confirm db error", svcErr: fmt.Errorf("x: %w", dbErr), call: callConfirmUpload(TEST_USER_ID, TEST_FILE_ID)},
		{name: "download bad user", call: callGetDownloadUrl(TEST_BAD_ID, TEST_FILE_ID)},
		{name: "download bad file", call: callGetDownloadUrl(TEST_USER_ID, TEST_BAD_ID)},
		{name: "download not found", svcErr: domain.ErrMetadataNotFound, call: callGetDownloadUrl(TEST_USER_ID, TEST_FILE_ID)},
		{name: "download not confirmed", svcErr: domain.ErrFileNotConfirmed, call: callGetDownloadUrl(TEST_USER_ID, TEST_FILE_ID)},
		{name: "download db error", svcErr: dbErr, call: callGetDownloadUrl(TEST_USER_ID, TEST_FILE_ID)},
		{name: "list bad user", call: callListFiles(TEST_BAD_ID)},
		{name: "list service status", svcErr: domain.ErrFileNotConfirmed, call: callListFiles(TEST_USER_ID)},
		{name: "list db error", svcErr: dbErr, call: callListFiles(TEST_USER_ID)},
		{name: "has bad user", call: callHasRequiredFiles(TEST_BAD_ID)},
		{name: "has service status", svcErr: domain.ErrFileIsEmpty, call: callHasRequiredFiles(TEST_USER_ID)},
		{name: "has db error", svcErr: dbErr, call: callHasRequiredFiles(TEST_USER_ID)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(t.Context(), newTestFileServer(failingFake(tt.svcErr)))
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok, "error is not a grpc status: %v", err)
			assert.Contains(t, allowed, st.Code())
		})
	}
}

func TestFileServer_InvariantParseErrorSkipsService(t *testing.T) {
	tests := []struct {
		name    string
		call    serverCall
		wantErr error
	}{
		{name: "init bad user", call: callInitUpload(TEST_BAD_ID, filestorev1.FileType_FILE_TYPE_PASSPORT), wantErr: domain.ErrUserIDMustBeUUID},
		{name: "init unspecified type", call: callInitUpload(TEST_USER_ID, filestorev1.FileType_FILE_TYPE_UNSPECIFIED), wantErr: domain.ErrFileTypeRequired},
		{name: "init unknown type", call: callInitUpload(TEST_USER_ID, filestorev1.FileType(4)), wantErr: domain.ErrFileTypeUnknown},
		{name: "confirm bad user", call: callConfirmUpload("", TEST_FILE_ID), wantErr: domain.ErrUserIDMustBeUUID},
		{name: "confirm bad file", call: callConfirmUpload(TEST_USER_ID, ""), wantErr: domain.ErrFileIDMustBeUUID},
		{name: "download bad user", call: callGetDownloadUrl("", TEST_FILE_ID), wantErr: domain.ErrUserIDMustBeUUID},
		{name: "download bad file", call: callGetDownloadUrl(TEST_USER_ID, ""), wantErr: domain.ErrFileIDMustBeUUID},
		{name: "list bad user", call: callListFiles(TEST_BAD_ID), wantErr: domain.ErrUserIDMustBeUUID},
		{name: "has bad user", call: callHasRequiredFiles(TEST_BAD_ID), wantErr: domain.ErrUserIDMustBeUUID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeFileStorageService{}
			err := tt.call(t.Context(), newTestFileServer(fake))
			assertStatusErr(t, err, tt.wantErr)
			assert.Equal(t, 0, fake.totalCalls())
		})
	}
}

func TestFileServer_InvariantInternalMessage(t *testing.T) {
	failedMessages := []string{
		status.Convert(domain.ErrInitUploadFailed).Message(),
		status.Convert(domain.ErrConfirmUploadFailed).Message(),
		status.Convert(domain.ErrDownloadURLFailed).Message(),
		status.Convert(domain.ErrListFilesFailed).Message(),
		status.Convert(domain.ErrRequiredCheckFailed).Message(),
	}
	methods := []struct {
		name    string
		call    serverCall
		wantErr error
	}{
		{name: "init", call: callInitUpload(TEST_USER_ID, filestorev1.FileType_FILE_TYPE_PASSPORT), wantErr: domain.ErrInitUploadFailed},
		{name: "confirm", call: callConfirmUpload(TEST_USER_ID, TEST_FILE_ID), wantErr: domain.ErrConfirmUploadFailed},
		{name: "download", call: callGetDownloadUrl(TEST_USER_ID, TEST_FILE_ID), wantErr: domain.ErrDownloadURLFailed},
		{name: "list", call: callListFiles(TEST_USER_ID), wantErr: domain.ErrListFilesFailed},
		{name: "has", call: callHasRequiredFiles(TEST_USER_ID), wantErr: domain.ErrRequiredCheckFailed},
	}
	svcErrors := []struct {
		name string
		err  error
	}{
		{name: "plain", err: errors.New(TEST_DB_ERROR_TEXT)},
		{name: "wrapped", err: fmt.Errorf("x: %w", errors.New(TEST_DB_ERROR_TEXT))},
		{name: "empty text", err: errors.New("")},
	}
	for _, m := range methods {
		for _, se := range svcErrors {
			t.Run(m.name+" "+se.name, func(t *testing.T) {
				err := m.call(t.Context(), newTestFileServer(failingFake(se.err)))
				assertStatusErr(t, err, m.wantErr)
				assert.Equal(t, codes.Internal, status.Code(err))
				assert.Contains(t, failedMessages, status.Convert(err).Message())
			})
		}
	}
}
