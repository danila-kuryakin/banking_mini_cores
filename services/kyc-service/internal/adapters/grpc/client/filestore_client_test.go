package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/pb/gen/filestore/v1"
)

// fileMeta собирает метаданные файла только с типом и статусом
func fileMeta(fileType filestorev1.FileType, fileStatus filestorev1.FileStatus) *filestorev1.FileMetadata {
	return &filestorev1.FileMetadata{Type: fileType, Status: fileStatus}
}

func TestFilestoreClient_HasRequiredDocuments(t *testing.T) {
	tests := []struct {
		name        string
		resp        *filestorev1.HasRequiredFilesResponse
		wantOK      bool
		wantMissing []models.DocumentType
	}{
		{
			name: "missing from spec example keeps order",
			resp: &filestorev1.HasRequiredFilesResponse{
				Ok:      false,
				Missing: []filestorev1.FileType{filestorev1.FileType_FILE_TYPE_SELFIE, filestorev1.FileType_FILE_TYPE_PASSPORT},
			},
			wantOK:      false,
			wantMissing: []models.DocumentType{models.DOCUMENT_TYPE_SELFIE, models.DOCUMENT_TYPE_PASSPORT},
		},
		{
			name: "all three types",
			resp: &filestorev1.HasRequiredFilesResponse{
				Ok: false,
				Missing: []filestorev1.FileType{
					filestorev1.FileType_FILE_TYPE_PROOF_OF_ADDRESS,
					filestorev1.FileType_FILE_TYPE_PASSPORT,
					filestorev1.FileType_FILE_TYPE_SELFIE,
				},
			},
			wantOK:      false,
			wantMissing: []models.DocumentType{models.DOCUMENT_TYPE_PROOF_OF_ADDRESS, models.DOCUMENT_TYPE_PASSPORT, models.DOCUMENT_TYPE_SELFIE},
		},
		{
			name:        "complete with nil missing",
			resp:        &filestorev1.HasRequiredFilesResponse{Ok: true, Missing: nil},
			wantOK:      true,
			wantMissing: []models.DocumentType{},
		},
		{
			name:        "complete with empty missing",
			resp:        &filestorev1.HasRequiredFilesResponse{Ok: true, Missing: []filestorev1.FileType{}},
			wantOK:      true,
			wantMissing: []models.DocumentType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := &fakeFileServiceClient{hasRequiredResp: tt.resp}
			filestoreClient := NewFilestoreClient(files)

			gotOK, gotMissing, err := filestoreClient.HasRequiredDocuments(t.Context(), TEST_USER_ID)

			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, gotOK)
			// порядок важен, поэтому сравниваем поэлементно; nil и пустой срез не различаем
			require.Len(t, gotMissing, len(tt.wantMissing))
			for i := range tt.wantMissing {
				assert.Equal(t, tt.wantMissing[i], gotMissing[i])
			}
			assertProtoCalls(t, []*filestorev1.HasRequiredFilesRequest{{UserId: TEST_USER_ID.String()}}, files.hasRequiredCalls)
		})
	}
}

func TestFilestoreClient_HasRequiredDocuments_UpstreamErrors(t *testing.T) {
	tests := []struct {
		name     string
		upstream error
		wantCode codes.Code
	}{
		{name: "unavailable", upstream: status.Error(codes.Unavailable, "connection refused"), wantCode: codes.Unavailable},
		{name: "not found", upstream: status.Error(codes.NotFound, "user not found"), wantCode: codes.NotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := &fakeFileServiceClient{hasRequiredErr: tt.upstream}
			filestoreClient := NewFilestoreClient(files)

			gotOK, _, err := filestoreClient.HasRequiredDocuments(t.Context(), TEST_USER_ID)

			require.ErrorIs(t, err, tt.upstream)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.False(t, gotOK)
		})
	}
}

func TestFilestoreClient_ConfirmedTypes(t *testing.T) {
	tests := []struct {
		name  string
		files []*filestorev1.FileMetadata
		want  []models.DocumentType
	}{
		{
			name: "mixed list from spec example",
			files: []*filestorev1.FileMetadata{
				fileMeta(filestorev1.FileType_FILE_TYPE_SELFIE, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
				fileMeta(filestorev1.FileType_FILE_TYPE_PASSPORT, filestorev1.FileStatus_FILE_STATUS_UPLOADED),
				fileMeta(filestorev1.FileType_FILE_TYPE_PASSPORT, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
				fileMeta(filestorev1.FileType_FILE_TYPE_SELFIE, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
				fileMeta(filestorev1.FileType_FILE_TYPE_UNSPECIFIED, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
				fileMeta(filestorev1.FileType_FILE_TYPE_PROOF_OF_ADDRESS, filestorev1.FileStatus_FILE_STATUS_UNSPECIFIED),
			},
			want: []models.DocumentType{models.DOCUMENT_TYPE_SELFIE, models.DOCUMENT_TYPE_PASSPORT},
		},
		{
			name:  "nil file list",
			files: nil,
			want:  []models.DocumentType{},
		},
		{
			name:  "empty file list",
			files: []*filestorev1.FileMetadata{},
			want:  []models.DocumentType{},
		},
		{
			name: "only unconfirmed files",
			files: []*filestorev1.FileMetadata{
				fileMeta(filestorev1.FileType_FILE_TYPE_PASSPORT, filestorev1.FileStatus_FILE_STATUS_UPLOADED),
				fileMeta(filestorev1.FileType_FILE_TYPE_SELFIE, filestorev1.FileStatus_FILE_STATUS_UNSPECIFIED),
			},
			want: []models.DocumentType{},
		},
		{
			name: "unknown and unspecified types skipped",
			files: []*filestorev1.FileMetadata{
				fileMeta(filestorev1.FileType(99), filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
				fileMeta(filestorev1.FileType_FILE_TYPE_UNSPECIFIED, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
				fileMeta(filestorev1.FileType_FILE_TYPE_PASSPORT, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
			},
			want: []models.DocumentType{models.DOCUMENT_TYPE_PASSPORT},
		},
		{
			name: "duplicates collapsed in first-seen order",
			files: []*filestorev1.FileMetadata{
				fileMeta(filestorev1.FileType_FILE_TYPE_PASSPORT, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
				fileMeta(filestorev1.FileType_FILE_TYPE_PASSPORT, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
				fileMeta(filestorev1.FileType_FILE_TYPE_SELFIE, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
				fileMeta(filestorev1.FileType_FILE_TYPE_PASSPORT, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
			},
			want: []models.DocumentType{models.DOCUMENT_TYPE_PASSPORT, models.DOCUMENT_TYPE_SELFIE},
		},
		{
			name: "all three types in given order",
			files: []*filestorev1.FileMetadata{
				fileMeta(filestorev1.FileType_FILE_TYPE_PROOF_OF_ADDRESS, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
				fileMeta(filestorev1.FileType_FILE_TYPE_SELFIE, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
				fileMeta(filestorev1.FileType_FILE_TYPE_PASSPORT, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
			},
			want: []models.DocumentType{models.DOCUMENT_TYPE_PROOF_OF_ADDRESS, models.DOCUMENT_TYPE_SELFIE, models.DOCUMENT_TYPE_PASSPORT},
		},
		{
			name: "unknown file status skipped",
			files: []*filestorev1.FileMetadata{
				fileMeta(filestorev1.FileType_FILE_TYPE_PASSPORT, filestorev1.FileStatus(99)),
				fileMeta(filestorev1.FileType_FILE_TYPE_SELFIE, filestorev1.FileStatus_FILE_STATUS_CONFIRMED),
			},
			want: []models.DocumentType{models.DOCUMENT_TYPE_SELFIE},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := &fakeFileServiceClient{listFilesResp: &filestorev1.ListFilesResponse{Files: tt.files}}
			filestoreClient := NewFilestoreClient(files)

			got, err := filestoreClient.ConfirmedTypes(t.Context(), TEST_USER_ID)

			require.NoError(t, err)
			// порядок важен, поэтому сравниваем поэлементно; nil и пустой срез не различаем
			require.Len(t, got, len(tt.want))
			for i := range tt.want {
				assert.Equal(t, tt.want[i], got[i])
			}
			assertProtoCalls(t, []*filestorev1.ListFilesRequest{{UserId: TEST_USER_ID.String(), Limit: domain.FILE_LIST_LIMIT}}, files.listFilesCalls)
		})
	}
}

func TestFilestoreClient_ConfirmedTypes_UpstreamErrors(t *testing.T) {
	tests := []struct {
		name     string
		upstream error
		wantCode codes.Code
	}{
		{name: "unavailable", upstream: status.Error(codes.Unavailable, "connection refused"), wantCode: codes.Unavailable},
		{name: "internal", upstream: status.Error(codes.Internal, "db is down"), wantCode: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := &fakeFileServiceClient{listFilesErr: tt.upstream}
			filestoreClient := NewFilestoreClient(files)

			got, err := filestoreClient.ConfirmedTypes(t.Context(), TEST_USER_ID)

			require.ErrorIs(t, err, tt.upstream)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.Nil(t, got)
		})
	}
}
