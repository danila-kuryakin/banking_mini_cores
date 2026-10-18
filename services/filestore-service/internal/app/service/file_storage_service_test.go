package service

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/adapters/storage_s3"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	TEST_PUT_URL = "http://minio.test/put"
	TEST_GET_URL = "http://minio.test/get"
)

// newTestService - юнит поверх фейков
func newTestService(repo *fakeMetadataRepo, storage *fakeMinIO) *FileStorageService {
	return NewFileStorageService(&repository.Repository{Metadata: repo}, &storage_s3.Storage{MinIO: storage})
}

// assertExpiresAt - срок ссылки в [before+ttl, after+ttl] и в UTC
func assertExpiresAt(t *testing.T, expiresAt, before, after time.Time, ttl time.Duration) {
	t.Helper()
	assert.WithinRange(t, expiresAt, before.Add(ttl), after.Add(ttl))
	assert.Equal(t, time.UTC, expiresAt.Location())
}

func TestInitUpload_CreateMetadataArgs(t *testing.T) {
	tests := []struct {
		name     string
		fileType models.FileType
		filename string
	}{
		{name: "passport with cyrillic filename", fileType: models.FILE_TYPE_PASSPORT, filename: "паспорт.pdf"},
		{name: "selfie with empty filename", fileType: models.FILE_TYPE_SELFIE, filename: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.MustParse(TEST_USER_ID)
			repo := &fakeMetadataRepo{createResult: &models.Metadata{ObjectPath: "x/y/z"}}
			storage := &fakeMinIO{putURL: TEST_PUT_URL}
			svc := newTestService(repo, storage)

			_, _, err := svc.InitUpload(t.Context(), userID, tt.fileType, tt.filename)

			require.NoError(t, err)
			require.Len(t, repo.createCalls, 1)
			created := repo.createCalls[0]
			assert.NotEqual(t, uuid.Nil, created.ID)
			assert.Equal(t, userID, created.UserID)
			assert.Equal(t, tt.fileType, created.Type)
			assert.Equal(t, tt.filename, created.Filename)
			assert.Equal(t, TEST_USER_ID+"/"+string(tt.fileType)+"/"+created.ID.String(), created.ObjectPath)
		})
	}
}

func TestInitUpload_DistinctIDs(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	repo := &fakeMetadataRepo{createResult: &models.Metadata{ObjectPath: "x/y/z"}}
	storage := &fakeMinIO{putURL: TEST_PUT_URL}
	svc := newTestService(repo, storage)

	_, _, err := svc.InitUpload(t.Context(), userID, models.FILE_TYPE_PASSPORT, "a.pdf")
	require.NoError(t, err)
	_, _, err = svc.InitUpload(t.Context(), userID, models.FILE_TYPE_PASSPORT, "a.pdf")
	require.NoError(t, err)

	require.Len(t, repo.createCalls, 2)
	assert.NotEqual(t, uuid.Nil, repo.createCalls[0].ID)
	assert.NotEqual(t, uuid.Nil, repo.createCalls[1].ID)
	assert.NotEqual(t, repo.createCalls[0].ID, repo.createCalls[1].ID)
}

func TestInitUpload_Errors(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	errDB := errors.New("db failure")
	errPresign := errors.New("presign failure")

	tests := []struct {
		name         string
		createErr    error
		putErr       error
		wantErr      error
		wantPutCalls []presignPutCall
	}{
		{
			name:      "create metadata failure",
			createErr: errDB,
			wantErr:   errDB,
		},
		{
			name:         "presigned put failure",
			putErr:       errPresign,
			wantErr:      errPresign,
			wantPutCalls: []presignPutCall{{ObjectPath: "x/y/z", Expiry: domain.UPLOAD_URL_TTL}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeMetadataRepo{createResult: &models.Metadata{ObjectPath: "x/y/z"}, createErr: tt.createErr}
			storage := &fakeMinIO{putErr: tt.putErr}
			svc := newTestService(repo, storage)

			metadata, url, err := svc.InitUpload(t.Context(), userID, models.FILE_TYPE_PASSPORT, "a.pdf")

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, metadata)
			assert.Nil(t, url)
			assert.Len(t, repo.createCalls, 1)
			assert.Equal(t, tt.wantPutCalls, storage.putCalls)
			assert.Nil(t, repo.getCalls)
			assert.Nil(t, repo.confirmCalls)
			assert.Nil(t, repo.listCalls)
			assert.Nil(t, repo.listConfirmedCalls)
		})
	}
}

func TestInitUpload_Success(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	// фейк вернул запись с другим ObjectPath - ссылка подписывается по ней
	created := &models.Metadata{ObjectPath: "x/y/z", Status: models.FILE_STATUS_UPLOADED}
	repo := &fakeMetadataRepo{createResult: created}
	storage := &fakeMinIO{putURL: TEST_PUT_URL}
	svc := newTestService(repo, storage)

	before := time.Now()
	metadata, url, err := svc.InitUpload(t.Context(), userID, models.FILE_TYPE_PASSPORT, "паспорт.pdf")
	after := time.Now()

	require.NoError(t, err)
	assert.Same(t, created, metadata)
	require.NotNil(t, url)
	assert.Equal(t, TEST_PUT_URL, url.URL)
	assertExpiresAt(t, url.ExpiresAt, before, after, domain.UPLOAD_URL_TTL)
	assert.Equal(t, []presignPutCall{{ObjectPath: "x/y/z", Expiry: domain.UPLOAD_URL_TTL}}, storage.putCalls)
}

func TestInitUpload_ObjectPathFormatInvariant(t *testing.T) {
	fileTypes := []models.FileType{models.FILE_TYPE_PASSPORT, models.FILE_TYPE_SELFIE, models.FILE_TYPE_PROOF_OF_ADDRESS}

	for _, fileType := range fileTypes {
		t.Run(string(fileType), func(t *testing.T) {
			userID := uuid.MustParse(TEST_USER_ID)
			repo := &fakeMetadataRepo{createResult: &models.Metadata{ObjectPath: "x/y/z"}}
			storage := &fakeMinIO{putURL: TEST_PUT_URL}
			svc := newTestService(repo, storage)

			_, _, err := svc.InitUpload(t.Context(), userID, fileType, "doc.pdf")

			require.NoError(t, err)
			require.Len(t, repo.createCalls, 1)
			created := repo.createCalls[0]
			assert.Equal(t, fmt.Sprintf("%s/%s/%s", userID, fileType, created.ID), created.ObjectPath)
		})
	}
}

func TestGetDownloadURL_Errors(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	fileID := uuid.MustParse(TEST_FILE_ID)
	errDB := errors.New("db failure")
	errPresign := errors.New("presign failure")
	confirmedOpts := []presignGetCall{{
		ObjectPath: "u/selfie/f",
		Expiry:     domain.DOWNLOAD_URL_TTL,
		Opts:       models.DownloadOptions{Filename: "селфи.jpg", ContentType: "image/jpeg"},
	}}

	tests := []struct {
		name            string
		recordStatus    models.FileStatus
		getErr          error
		getURLErr       error
		wantErr         error
		wantCode        codes.Code
		wantGetURLCalls []presignGetCall
	}{
		{name: "metadata not found", getErr: domain.ErrMetadataNotFound, wantErr: domain.ErrMetadataNotFound, wantCode: codes.Unknown},
		{name: "metadata repository failure", getErr: errDB, wantErr: errDB, wantCode: codes.Unknown},
		{name: "status uploaded", recordStatus: models.FILE_STATUS_UPLOADED, wantErr: domain.ErrFileNotConfirmed, wantCode: codes.FailedPrecondition},
		{name: "status empty", recordStatus: models.FileStatus(""), wantErr: domain.ErrFileNotConfirmed, wantCode: codes.FailedPrecondition},
		{name: "status unknown", recordStatus: models.FileStatus("deleted"), wantErr: domain.ErrFileNotConfirmed, wantCode: codes.FailedPrecondition},
		{
			name:            "presigned get failure",
			recordStatus:    models.FILE_STATUS_CONFIRMED,
			getURLErr:       errPresign,
			wantErr:         errPresign,
			wantCode:        codes.Unknown,
			wantGetURLCalls: confirmedOpts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &models.Metadata{
				ID:          fileID,
				UserID:      userID,
				Type:        models.FILE_TYPE_SELFIE,
				Status:      tt.recordStatus,
				ObjectPath:  "u/selfie/f",
				Filename:    "селфи.jpg",
				ContentType: "image/jpeg",
			}
			repo := &fakeMetadataRepo{getResult: record, getErr: tt.getErr}
			storage := &fakeMinIO{getURLErr: tt.getURLErr}
			svc := newTestService(repo, storage)

			url, err := svc.GetDownloadURL(t.Context(), userID, fileID)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.Nil(t, url)
			assert.Equal(t, []getMetadataCall{{FileID: fileID, UserID: userID}}, repo.getCalls)
			assert.Equal(t, tt.wantGetURLCalls, storage.getURLCalls)
		})
	}
}

func TestGetDownloadURL_Success(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		contentType string
	}{
		{name: "cyrillic filename", filename: "селфи.jpg", contentType: "image/jpeg"},
		{name: "empty filename", filename: "", contentType: "application/pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.MustParse(TEST_USER_ID)
			fileID := uuid.MustParse(TEST_FILE_ID)
			record := &models.Metadata{
				ID:          fileID,
				UserID:      userID,
				Type:        models.FILE_TYPE_SELFIE,
				Status:      models.FILE_STATUS_CONFIRMED,
				ObjectPath:  "u/selfie/f",
				Filename:    tt.filename,
				ContentType: tt.contentType,
			}
			repo := &fakeMetadataRepo{getResult: record}
			storage := &fakeMinIO{getURL: TEST_GET_URL}
			svc := newTestService(repo, storage)

			before := time.Now()
			url, err := svc.GetDownloadURL(t.Context(), userID, fileID)
			after := time.Now()

			require.NoError(t, err)
			require.NotNil(t, url)
			assert.Equal(t, TEST_GET_URL, url.URL)
			assertExpiresAt(t, url.ExpiresAt, before, after, domain.DOWNLOAD_URL_TTL)
			assert.Equal(t, []getMetadataCall{{FileID: fileID, UserID: userID}}, repo.getCalls)
			assert.Equal(t, []presignGetCall{{
				ObjectPath: "u/selfie/f",
				Expiry:     domain.DOWNLOAD_URL_TTL,
				Opts:       models.DownloadOptions{Filename: tt.filename, ContentType: tt.contentType},
			}}, storage.getURLCalls)
		})
	}
}

func TestListFiles_Pagination(t *testing.T) {
	tests := []struct {
		name       string
		limit      int
		offset     int
		wantLimit  int
		wantOffset int
	}{
		{name: "zero limit and offset", limit: 0, offset: 0, wantLimit: 20, wantOffset: 0},
		{name: "negative limit and offset", limit: -5, offset: -3, wantLimit: 20, wantOffset: 0},
		{name: "min int limit and offset", limit: math.MinInt, offset: math.MinInt, wantLimit: 20, wantOffset: 0},
		{name: "limit one", limit: 1, offset: 0, wantLimit: 1, wantOffset: 0},
		{name: "default size passed through", limit: 20, offset: 40, wantLimit: 20, wantOffset: 40},
		{name: "limit 99", limit: 99, offset: 0, wantLimit: 99, wantOffset: 0},
		{name: "limit max", limit: 100, offset: 7, wantLimit: 100, wantOffset: 7},
		{name: "limit max plus one", limit: 101, offset: 0, wantLimit: 100, wantOffset: 0},
		{name: "limit max int", limit: math.MaxInt, offset: 0, wantLimit: 100, wantOffset: 0},
		{name: "offset minus one", limit: 10, offset: -1, wantLimit: 10, wantOffset: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.MustParse(TEST_USER_ID)
			repo := &fakeMetadataRepo{listResult: []*models.Metadata{}}
			svc := newTestService(repo, &fakeMinIO{})

			_, err := svc.ListFiles(t.Context(), userID, tt.limit, tt.offset)

			require.NoError(t, err)
			assert.Equal(t, []listMetadataCall{{UserID: userID, Limit: tt.wantLimit, Offset: tt.wantOffset}}, repo.listCalls)
		})
	}
}

func TestListFiles_Result(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	errDB := errors.New("db failure")
	twoFiles := []*models.Metadata{
		{ID: uuid.MustParse(TEST_FILE_ID), UserID: userID, Type: models.FILE_TYPE_PASSPORT},
		{ID: uuid.MustParse("33333333-3333-3333-3333-333333333333"), UserID: userID, Type: models.FILE_TYPE_SELFIE},
	}

	tests := []struct {
		name       string
		listResult []*models.Metadata
		listErr    error
		want       []*models.Metadata
		wantErr    error
	}{
		{name: "two files", listResult: twoFiles, want: twoFiles},
		{name: "empty non-nil slice", listResult: []*models.Metadata{}, want: []*models.Metadata{}},
		{name: "repository failure", listErr: errDB, wantErr: errDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeMetadataRepo{listResult: tt.listResult, listErr: tt.listErr}
			svc := newTestService(repo, &fakeMinIO{})

			got, err := svc.ListFiles(t.Context(), userID, 20, 0)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			// сравнение и по значению, и по типу: пустой не-nil срез не должен стать nil
			assert.Equal(t, tt.want, got)
			require.Len(t, got, len(tt.want))
			for i := range tt.want {
				assert.Same(t, tt.want[i], got[i])
			}
		})
	}
}

func TestListFiles_PageBoundsInvariant(t *testing.T) {
	limits := []int{math.MinInt, -1, 0, 1, 50, 100, 101, math.MaxInt}
	offsets := []int{math.MinInt, -1, 0, 1, math.MaxInt}

	for _, limit := range limits {
		for _, offset := range offsets {
			t.Run(fmt.Sprintf("limit=%d offset=%d", limit, offset), func(t *testing.T) {
				repo := &fakeMetadataRepo{listResult: []*models.Metadata{}}
				svc := newTestService(repo, &fakeMinIO{})

				_, err := svc.ListFiles(t.Context(), uuid.MustParse(TEST_USER_ID), limit, offset)

				require.NoError(t, err)
				require.Len(t, repo.listCalls, 1)
				assert.GreaterOrEqual(t, repo.listCalls[0].Limit, 1)
				assert.LessOrEqual(t, repo.listCalls[0].Limit, 100)
				assert.GreaterOrEqual(t, repo.listCalls[0].Offset, 0)
			})
		}
	}
}

// hasRequiredCases - ответы ListConfirmedTypes из п.6 спецификации B
func hasRequiredCases() []struct {
	name        string
	confirmed   []models.FileType
	wantOK      bool
	wantMissing []models.FileType
} {
	passport := models.FILE_TYPE_PASSPORT
	selfie := models.FILE_TYPE_SELFIE
	proof := models.FILE_TYPE_PROOF_OF_ADDRESS

	return []struct {
		name        string
		confirmed   []models.FileType
		wantOK      bool
		wantMissing []models.FileType
	}{
		{name: "nil", confirmed: nil, wantOK: false, wantMissing: []models.FileType{passport, selfie}},
		{name: "empty", confirmed: []models.FileType{}, wantOK: false, wantMissing: []models.FileType{passport, selfie}},
		{name: "selfie only", confirmed: []models.FileType{selfie}, wantOK: false, wantMissing: []models.FileType{passport}},
		{name: "passport only", confirmed: []models.FileType{passport}, wantOK: false, wantMissing: []models.FileType{selfie}},
		{name: "both reversed", confirmed: []models.FileType{selfie, passport}, wantOK: true, wantMissing: []models.FileType{}},
		{name: "both and proof of address", confirmed: []models.FileType{passport, selfie, proof}, wantOK: true, wantMissing: []models.FileType{}},
		{name: "proof of address only", confirmed: []models.FileType{proof}, wantOK: false, wantMissing: []models.FileType{passport, selfie}},
		{name: "passport twice", confirmed: []models.FileType{passport, passport}, wantOK: false, wantMissing: []models.FileType{selfie}},
	}
}

// nonNilTypes - nil и пустой срез missing не различаются (R15)
func nonNilTypes(types []models.FileType) []models.FileType {
	return append([]models.FileType{}, types...)
}

func TestHasRequiredFiles(t *testing.T) {
	for _, tt := range hasRequiredCases() {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.MustParse(TEST_USER_ID)
			repo := &fakeMetadataRepo{confirmedTypes: tt.confirmed}
			svc := newTestService(repo, &fakeMinIO{})

			ok, missing, err := svc.HasRequiredFiles(t.Context(), userID)

			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, nonNilTypes(tt.wantMissing), nonNilTypes(missing))
			assert.Equal(t, []uuid.UUID{userID}, repo.listConfirmedCalls)
		})
	}
}

func TestHasRequiredFiles_RepositoryError(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	errDB := errors.New("db failure")
	repo := &fakeMetadataRepo{listConfirmedErr: errDB}
	svc := newTestService(repo, &fakeMinIO{})

	ok, missing, err := svc.HasRequiredFiles(t.Context(), userID)

	require.ErrorIs(t, err, errDB)
	assert.False(t, ok)
	assert.Nil(t, missing)
	assert.Equal(t, []uuid.UUID{userID}, repo.listConfirmedCalls)
}

func TestHasRequiredFiles_MissingInvariant(t *testing.T) {
	for _, tt := range hasRequiredCases() {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeMetadataRepo{confirmedTypes: tt.confirmed}
			svc := newTestService(repo, &fakeMinIO{})

			ok, missing, err := svc.HasRequiredFiles(t.Context(), uuid.MustParse(TEST_USER_ID))

			require.NoError(t, err)
			assert.Subset(t, models.REQUIRED_FILE_TYPES, missing)
			unique := make(map[models.FileType]struct{}, len(missing))
			for _, fileType := range missing {
				unique[fileType] = struct{}{}
			}
			assert.Len(t, unique, len(missing))
			assert.Equal(t, len(missing) == 0, ok)
		})
	}
}
