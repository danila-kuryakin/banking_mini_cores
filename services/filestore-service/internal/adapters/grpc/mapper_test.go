package grpc

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/pb/gen/filestore/v1"
)

// общие тестовые идентификаторы пакета
const (
	TEST_USER_ID     = "11111111-1111-1111-1111-111111111111"
	TEST_FILE_ID     = "33333333-3333-3333-3333-333333333333"
	TEST_NEW_FILE_ID = "44444444-4444-4444-4444-444444444444"
	TEST_BAD_ID      = "abc"
	TEST_NIL_UUID    = "00000000-0000-0000-0000-000000000000"
)

// доменные значения типов и статусов
const (
	TEST_TYPE_PASSPORT         models.FileType   = "passport"
	TEST_TYPE_SELFIE           models.FileType   = "selfie"
	TEST_TYPE_PROOF_OF_ADDRESS models.FileType   = "proof_of_address"
	TEST_STATUS_UPLOADED       models.FileStatus = "uploaded"
	TEST_STATUS_CONFIRMED      models.FileStatus = "confirmed"
)

func TestTypeToProto(t *testing.T) {
	tests := []struct {
		name string
		in   models.FileType
		want filestorev1.FileType
	}{
		{name: "passport", in: TEST_TYPE_PASSPORT, want: filestorev1.FileType_FILE_TYPE_PASSPORT},
		{name: "selfie", in: TEST_TYPE_SELFIE, want: filestorev1.FileType_FILE_TYPE_SELFIE},
		{name: "proof_of_address", in: TEST_TYPE_PROOF_OF_ADDRESS, want: filestorev1.FileType_FILE_TYPE_PROOF_OF_ADDRESS},
		{name: "empty", in: "", want: filestorev1.FileType_FILE_TYPE_UNSPECIFIED},
		{name: "upper case", in: "PASSPORT", want: filestorev1.FileType_FILE_TYPE_UNSPECIFIED},
		{name: "unknown", in: "unknown", want: filestorev1.FileType_FILE_TYPE_UNSPECIFIED},
		{name: "leading space", in: " passport", want: filestorev1.FileType_FILE_TYPE_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, typeToProto(tt.in))
		})
	}
}

func TestTypeFromProto(t *testing.T) {
	tests := []struct {
		name string
		in   filestorev1.FileType
		want models.FileType
	}{
		{name: "passport", in: filestorev1.FileType_FILE_TYPE_PASSPORT, want: TEST_TYPE_PASSPORT},
		{name: "selfie", in: filestorev1.FileType_FILE_TYPE_SELFIE, want: TEST_TYPE_SELFIE},
		{name: "proof_of_address", in: filestorev1.FileType_FILE_TYPE_PROOF_OF_ADDRESS, want: TEST_TYPE_PROOF_OF_ADDRESS},
		{name: "unspecified", in: filestorev1.FileType_FILE_TYPE_UNSPECIFIED, want: ""},
		{name: "out of enum 4", in: filestorev1.FileType(4), want: ""},
		{name: "out of enum 99", in: filestorev1.FileType(99), want: ""},
		{name: "negative", in: filestorev1.FileType(-1), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, typeFromProto(tt.in))
		})
	}
}

func TestStatusToProto(t *testing.T) {
	tests := []struct {
		name string
		in   models.FileStatus
		want filestorev1.FileStatus
	}{
		{name: "uploaded", in: TEST_STATUS_UPLOADED, want: filestorev1.FileStatus_FILE_STATUS_UPLOADED},
		{name: "confirmed", in: TEST_STATUS_CONFIRMED, want: filestorev1.FileStatus_FILE_STATUS_CONFIRMED},
		{name: "empty", in: "", want: filestorev1.FileStatus_FILE_STATUS_UNSPECIFIED},
		{name: "upper case", in: "UPLOADED", want: filestorev1.FileStatus_FILE_STATUS_UNSPECIFIED},
		{name: "unknown", in: "deleted", want: filestorev1.FileStatus_FILE_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, statusToProto(tt.in))
		})
	}
}

func TestTimestampToProto_Nil(t *testing.T) {
	assert.Nil(t, timestampToProto(nil))
}

func TestTimestampToProto(t *testing.T) {
	// 2026-01-01 = 1767225600; плюс 86400 + 11045 = 97445
	utc := time.Date(2026, 1, 2, 3, 4, 5, 123456789, time.UTC)
	tests := []struct {
		name        string
		in          time.Time
		wantSeconds int64
		wantNanos   int32
	}{
		{name: "utc with nanos", in: utc, wantSeconds: 1767323045, wantNanos: 123456789},
		{name: "same moment in utc+3", in: utc.In(time.FixedZone("MSK", 3*60*60)), wantSeconds: 1767323045, wantNanos: 123456789},
		{name: "unix epoch", in: time.Unix(0, 0), wantSeconds: 0, wantNanos: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			got := timestampToProto(&in)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantSeconds, got.GetSeconds())
			assert.Equal(t, tt.wantNanos, got.GetNanos())
			assert.True(t, got.AsTime().Equal(tt.in))
		})
	}
}

func TestMetadataToProto(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	confirmedAt := time.Date(2026, 1, 2, 3, 5, 0, 0, time.UTC)
	tests := []struct {
		name string
		in   *models.Metadata
		want *filestorev1.FileMetadata
	}{
		{
			name: "confirmed file",
			in: &models.Metadata{
				ID:          uuid.MustParse(TEST_FILE_ID),
				UserID:      uuid.MustParse(TEST_USER_ID),
				Type:        TEST_TYPE_SELFIE,
				Status:      TEST_STATUS_CONFIRMED,
				ObjectPath:  "u/selfie/f",
				Filename:    "селфи.jpg",
				ContentType: "image/jpeg",
				SizeBytes:   2048,
				SHA256:      "abc123",
				CreatedAt:   createdAt,
				ConfirmedAt: &confirmedAt,
			},
			want: &filestorev1.FileMetadata{
				Id:          TEST_FILE_ID,
				UserId:      TEST_USER_ID,
				Type:        filestorev1.FileType_FILE_TYPE_SELFIE,
				Status:      filestorev1.FileStatus_FILE_STATUS_CONFIRMED,
				ContentType: "image/jpeg",
				SizeBytes:   2048,
				Sha256:      "abc123",
				CreatedAt:   timestamppb.New(createdAt),
				ConfirmedAt: timestamppb.New(confirmedAt),
				Filename:    "селфи.jpg",
			},
		},
		{
			name: "just uploaded file",
			in: &models.Metadata{
				ID:        uuid.MustParse(TEST_FILE_ID),
				UserID:    uuid.MustParse(TEST_USER_ID),
				Type:      TEST_TYPE_PASSPORT,
				Status:    TEST_STATUS_UPLOADED,
				CreatedAt: createdAt,
			},
			want: &filestorev1.FileMetadata{
				Id:        TEST_FILE_ID,
				UserId:    TEST_USER_ID,
				Type:      filestorev1.FileType_FILE_TYPE_PASSPORT,
				Status:    filestorev1.FileStatus_FILE_STATUS_UPLOADED,
				CreatedAt: timestamppb.New(createdAt),
			},
		},
		{
			name: "zero metadata",
			in:   &models.Metadata{},
			want: &filestorev1.FileMetadata{
				Id:     TEST_NIL_UUID,
				UserId: TEST_NIL_UUID,
				Type:   filestorev1.FileType_FILE_TYPE_UNSPECIFIED,
				Status: filestorev1.FileStatus_FILE_STATUS_UNSPECIFIED,
				// нулевое время - тот же момент, а не nil
				CreatedAt: timestamppb.New(time.Time{}),
			},
		},
		{
			name: "unknown type and status",
			in: &models.Metadata{
				ID:          uuid.MustParse(TEST_FILE_ID),
				UserID:      uuid.MustParse(TEST_USER_ID),
				Type:        "unknown",
				Status:      "deleted",
				Filename:    "a.pdf",
				ContentType: "application/pdf",
				SizeBytes:   1,
				SHA256:      "ff",
				CreatedAt:   createdAt,
			},
			want: &filestorev1.FileMetadata{
				Id:          TEST_FILE_ID,
				UserId:      TEST_USER_ID,
				Type:        filestorev1.FileType_FILE_TYPE_UNSPECIFIED,
				Status:      filestorev1.FileStatus_FILE_STATUS_UNSPECIFIED,
				ContentType: "application/pdf",
				SizeBytes:   1,
				Sha256:      "ff",
				CreatedAt:   timestamppb.New(createdAt),
				Filename:    "a.pdf",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metadataToProto(tt.in)
			require.NotNil(t, got)
			assert.True(t, proto.Equal(tt.want, got), "want %v, got %v", tt.want, got)
			assert.Equal(t, tt.in.ConfirmedAt == nil, got.GetConfirmedAt() == nil)
			assert.True(t, got.GetCreatedAt().AsTime().Equal(tt.in.CreatedAt))
		})
	}
}

func TestMetadataToProto_ObjectPathNotLeaked(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	base := models.Metadata{
		ID:        uuid.MustParse(TEST_FILE_ID),
		UserID:    uuid.MustParse(TEST_USER_ID),
		Type:      TEST_TYPE_SELFIE,
		Status:    TEST_STATUS_UPLOADED,
		CreatedAt: createdAt,
	}
	tests := []struct {
		name       string
		objectPath string
	}{
		{name: "empty path", objectPath: ""},
		{name: "real path", objectPath: "u/selfie/f"},
		{name: "other path", objectPath: TEST_USER_ID + "/selfie/" + TEST_FILE_ID},
	}
	want := metadataToProto(&base)
	require.NotNil(t, want)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := base
			in.ObjectPath = tt.objectPath
			assert.True(t, proto.Equal(want, metadataToProto(&in)))
		})
	}
}

func TestInvariant_TypeRoundTripFromDomain(t *testing.T) {
	tests := []struct {
		name string
		in   models.FileType
	}{
		{name: "passport", in: TEST_TYPE_PASSPORT},
		{name: "selfie", in: TEST_TYPE_SELFIE},
		{name: "proof_of_address", in: TEST_TYPE_PROOF_OF_ADDRESS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.in, typeFromProto(typeToProto(tt.in)))
		})
	}
}

func TestInvariant_TypeRoundTripFromProto(t *testing.T) {
	tests := []struct {
		name string
		in   filestorev1.FileType
	}{
		{name: "passport", in: filestorev1.FileType_FILE_TYPE_PASSPORT},
		{name: "selfie", in: filestorev1.FileType_FILE_TYPE_SELFIE},
		{name: "proof_of_address", in: filestorev1.FileType_FILE_TYPE_PROOF_OF_ADDRESS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.in, typeToProto(typeFromProto(tt.in)))
		})
	}
}

func TestInvariant_TypeToProtoInEnum(t *testing.T) {
	tests := []struct {
		name string
		in   models.FileType
	}{
		{name: "passport", in: TEST_TYPE_PASSPORT},
		{name: "selfie", in: TEST_TYPE_SELFIE},
		{name: "proof_of_address", in: TEST_TYPE_PROOF_OF_ADDRESS},
		{name: "empty", in: ""},
		{name: "upper case", in: "PASSPORT"},
		{name: "unknown", in: "unknown"},
		{name: "leading space", in: " passport"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, filestorev1.FileType_name, int32(typeToProto(tt.in)))
		})
	}
}
