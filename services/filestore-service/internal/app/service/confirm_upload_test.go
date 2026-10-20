package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

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
	TEST_USER_ID     = "11111111-1111-1111-1111-111111111111"
	TEST_FILE_ID     = "22222222-2222-2222-2222-222222222222"
	TEST_OBJECT_PATH = "u/passport/f"

	TEST_BODY_PDF  = "%PDF-1.4\n%test\n"
	TEST_BODY_PNG  = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"
	TEST_BODY_JPEG = "\xff\xd8\xff\xe0\x00\x10JFIF\x00"
	TEST_BODY_TEXT = "hello world"
	TEST_BODY_GIF  = "GIF89a"
	TEST_BODY_HTML = "<html><body>"

	TEST_SHA256_PDF  = "9d636b97713c8962c840e079a81f4805526bd2e3a1333bde969230f392a410f7"
	TEST_SHA256_PNG  = "02a3e298f1533f62558c58e4c70edcab9af5a50d62d925fd5390942020fb0fb8"
	TEST_SHA256_JPEG = "23e5c96c789570b1a740a7463526bb846d97506642e12a6a5e6b9b3b7a90cd5f"
)

// sha256Hex - эталонный sha256 из stdlib в hex нижнего регистра
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// paddedPDF - "%PDF-1.4" и нули до size байт
func paddedPDF(size int) []byte {
	data := make([]byte, size)
	copy(data, "%PDF-1.4")
	return data
}

// bodyOf - конструктор тела с заданными байтами
func bodyOf(data []byte) func() *fakeBody {
	return func() *fakeBody { return &fakeBody{data: data} }
}

// confirmSetup - настройка зависимостей для одного вызова ConfirmUpload по записи в uploaded
type confirmSetup struct {
	getErr       error
	statInfo     models.ObjectInfo
	statErr      error
	newBody      func() *fakeBody
	getObjectErr error
	removeErr    error
	confirmErr   error
}

// confirmFixture - собранный юнит и его фейки
type confirmFixture struct {
	svc           *FileStorageService
	repo          *fakeMetadataRepo
	storage       *fakeMinIO
	body          *fakeBody
	userID        uuid.UUID
	fileID        uuid.UUID
	confirmResult *models.Metadata
}

func newConfirmFixture(setup confirmSetup) confirmFixture {
	userID := uuid.MustParse(TEST_USER_ID)
	fileID := uuid.MustParse(TEST_FILE_ID)
	record := &models.Metadata{
		ID:         fileID,
		UserID:     userID,
		Type:       models.FILE_TYPE_PASSPORT,
		Status:     models.FILE_STATUS_UPLOADED,
		ObjectPath: TEST_OBJECT_PATH,
		Filename:   "passport.pdf",
	}
	confirmResult := &models.Metadata{
		ID:         fileID,
		UserID:     userID,
		Type:       models.FILE_TYPE_PASSPORT,
		Status:     models.FILE_STATUS_CONFIRMED,
		ObjectPath: TEST_OBJECT_PATH,
		Filename:   "passport.pdf",
	}
	body := setup.newBody()
	repo := &fakeMetadataRepo{
		getResult:     record,
		getErr:        setup.getErr,
		confirmResult: confirmResult,
		confirmErr:    setup.confirmErr,
	}
	storage := &fakeMinIO{
		statInfo:     setup.statInfo,
		statErr:      setup.statErr,
		body:         body,
		getObjectErr: setup.getObjectErr,
		removeErr:    setup.removeErr,
	}
	svc := NewFileStorageService(&repository.Repository{Metadata: repo}, &storage_s3.Storage{MinIO: storage})
	return confirmFixture{
		svc:           svc,
		repo:          repo,
		storage:       storage,
		body:          body,
		userID:        userID,
		fileID:        fileID,
		confirmResult: confirmResult,
	}
}

// assertNoUnrelatedCalls - ConfirmUpload не трогает методы, которые ему не нужны
func assertNoUnrelatedCalls(t *testing.T, repo *fakeMetadataRepo, storage *fakeMinIO) {
	t.Helper()
	assert.Nil(t, repo.createCalls)
	assert.Nil(t, repo.listCalls)
	assert.Nil(t, repo.listConfirmedCalls)
	assert.Nil(t, storage.putCalls)
	assert.Nil(t, storage.getURLCalls)
}

func TestConfirmUpload_Success(t *testing.T) {
	bigMax := paddedPDF(domain.MAX_FILE_SIZE)

	tests := []struct {
		name            string
		statInfo        models.ObjectInfo
		body            []byte
		chunk           int
		wantContentType string
		wantSize        int64
		wantSHA         string
	}{
		{
			name:            "pdf with stat equal to body",
			statInfo:        models.ObjectInfo{Key: TEST_OBJECT_PATH, Size: 15},
			body:            []byte(TEST_BODY_PDF),
			wantContentType: "application/pdf",
			wantSize:        15,
			wantSHA:         TEST_SHA256_PDF,
		},
		{
			name:            "png with stat larger than body",
			statInfo:        models.ObjectInfo{Key: TEST_OBJECT_PATH, Size: 100},
			body:            []byte(TEST_BODY_PNG),
			wantContentType: "image/png",
			wantSize:        16,
			wantSHA:         TEST_SHA256_PNG,
		},
		{
			name:            "jpeg",
			statInfo:        models.ObjectInfo{Key: TEST_OBJECT_PATH, Size: 11},
			body:            []byte(TEST_BODY_JPEG),
			wantContentType: "image/jpeg",
			wantSize:        11,
			wantSHA:         TEST_SHA256_JPEG,
		},
		{
			name:            "declared content type is ignored",
			statInfo:        models.ObjectInfo{Key: TEST_OBJECT_PATH, Size: 15, ContentType: "image/png"},
			body:            []byte(TEST_BODY_PDF),
			wantContentType: "application/pdf",
			wantSize:        15,
			wantSHA:         TEST_SHA256_PDF,
		},
		{
			name:            "stat size one byte, body larger",
			statInfo:        models.ObjectInfo{Key: TEST_OBJECT_PATH, Size: 1},
			body:            []byte(TEST_BODY_PDF),
			wantContentType: "application/pdf",
			wantSize:        15,
			wantSHA:         TEST_SHA256_PDF,
		},
		{
			name:            "body exactly max size",
			statInfo:        models.ObjectInfo{Key: TEST_OBJECT_PATH, Size: domain.MAX_FILE_SIZE},
			body:            bigMax,
			wantContentType: "application/pdf",
			wantSize:        domain.MAX_FILE_SIZE,
			wantSHA:         sha256Hex(bigMax),
		},
		{
			name:            "pdf delivered one byte per read",
			statInfo:        models.ObjectInfo{Key: TEST_OBJECT_PATH, Size: 15},
			body:            []byte(TEST_BODY_PDF),
			chunk:           1,
			wantContentType: "application/pdf",
			wantSize:        15,
			wantSHA:         TEST_SHA256_PDF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConfirmFixture(confirmSetup{
				statInfo: tt.statInfo,
				newBody:  func() *fakeBody { return &fakeBody{data: tt.body, chunk: tt.chunk} },
			})

			got, err := f.svc.ConfirmUpload(t.Context(), f.userID, f.fileID)

			require.NoError(t, err)
			assert.Same(t, f.confirmResult, got)
			assert.Equal(t, []getMetadataCall{{FileID: f.fileID, UserID: f.userID}}, f.repo.getCalls)
			assert.Equal(t, []string{TEST_OBJECT_PATH}, f.storage.statCalls)
			assert.Equal(t, []string{TEST_OBJECT_PATH}, f.storage.getObjectCalls)
			assert.Equal(t, []confirmMetadataCall{{
				FileID:      f.fileID,
				UserID:      f.userID,
				ContentType: tt.wantContentType,
				SizeBytes:   tt.wantSize,
				SHA256:      tt.wantSHA,
			}}, f.repo.confirmCalls)
			assert.Nil(t, f.storage.removeCalls)
			assert.True(t, f.body.closed)
			assertNoUnrelatedCalls(t, f.repo, f.storage)
		})
	}
}

func TestConfirmUpload_AlreadyConfirmed(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	fileID := uuid.MustParse(TEST_FILE_ID)
	record := &models.Metadata{
		ID:         fileID,
		UserID:     userID,
		Type:       models.FILE_TYPE_PASSPORT,
		Status:     models.FILE_STATUS_CONFIRMED,
		ObjectPath: TEST_OBJECT_PATH,
	}
	repo := &fakeMetadataRepo{getResult: record}
	storage := &fakeMinIO{}
	svc := NewFileStorageService(&repository.Repository{Metadata: repo}, &storage_s3.Storage{MinIO: storage})

	got, err := svc.ConfirmUpload(t.Context(), userID, fileID)

	require.NoError(t, err)
	assert.Same(t, record, got)
	assert.Equal(t, []getMetadataCall{{FileID: fileID, UserID: userID}}, repo.getCalls)
	assert.Nil(t, repo.confirmCalls)
	assert.Nil(t, storage.statCalls)
	assert.Nil(t, storage.getObjectCalls)
	assert.Nil(t, storage.removeCalls)
	assertNoUnrelatedCalls(t, repo, storage)
}

func TestConfirmUpload_Errors(t *testing.T) {
	userID := uuid.MustParse(TEST_USER_ID)
	fileID := uuid.MustParse(TEST_FILE_ID)
	errDB := errors.New("db failure")
	errStat := errors.New("stat failure")
	errGet := errors.New("get object failure")
	errRead := errors.New("read failure")
	errRemove := errors.New("remove failure")
	pathOnce := []string{TEST_OBJECT_PATH}
	statOK := models.ObjectInfo{Key: TEST_OBJECT_PATH, Size: 100}
	noBody := bodyOf(nil)
	// ConfirmMetadata для PDF из 15 байт
	pdfConfirm := []confirmMetadataCall{{
		FileID: fileID, UserID: userID, ContentType: "application/pdf", SizeBytes: 15, SHA256: TEST_SHA256_PDF,
	}}

	tests := []struct {
		name               string
		setup              confirmSetup
		wantErr            error
		wantCode           codes.Code
		wantNotErr         error
		wantStatCalls      []string
		wantGetObjectCalls []string
		wantRemoveCalls    []string
		wantConfirmCalls   []confirmMetadataCall
		wantClosed         bool
	}{
		{
			name:     "metadata not found",
			setup:    confirmSetup{getErr: domain.ErrMetadataNotFound, newBody: noBody},
			wantErr:  domain.ErrMetadataNotFound,
			wantCode: codes.Unknown,
		},
		{
			name:     "metadata repository failure",
			setup:    confirmSetup{getErr: errDB, newBody: noBody},
			wantErr:  errDB,
			wantCode: codes.Unknown,
		},
		{
			name:          "stat object not found",
			setup:         confirmSetup{statErr: domain.ErrObjectNotFound, newBody: noBody},
			wantErr:       domain.ErrFileNotUploaded,
			wantCode:      codes.FailedPrecondition,
			wantStatCalls: pathOnce,
		},
		{
			name:          "stat failure",
			setup:         confirmSetup{statErr: errStat, newBody: noBody},
			wantErr:       errStat,
			wantCode:      codes.Unknown,
			wantNotErr:    domain.ErrFileNotUploaded,
			wantStatCalls: pathOnce,
		},
		{
			name:            "stat size zero",
			setup:           confirmSetup{statInfo: models.ObjectInfo{Size: 0}, newBody: noBody},
			wantErr:         domain.ErrFileIsEmpty,
			wantCode:        codes.InvalidArgument,
			wantStatCalls:   pathOnce,
			wantRemoveCalls: pathOnce,
		},
		{
			name:            "stat size zero, remove fails",
			setup:           confirmSetup{statInfo: models.ObjectInfo{Size: 0}, newBody: noBody, removeErr: errRemove},
			wantErr:         domain.ErrFileIsEmpty,
			wantCode:        codes.InvalidArgument,
			wantStatCalls:   pathOnce,
			wantRemoveCalls: pathOnce,
		},
		{
			name:            "stat size max plus one",
			setup:           confirmSetup{statInfo: models.ObjectInfo{Size: domain.MAX_FILE_SIZE + 1}, newBody: noBody},
			wantErr:         domain.ErrFileIsTooLarge,
			wantCode:        codes.InvalidArgument,
			wantStatCalls:   pathOnce,
			wantRemoveCalls: pathOnce,
		},
		{
			name:            "stat size max plus one, remove fails",
			setup:           confirmSetup{statInfo: models.ObjectInfo{Size: domain.MAX_FILE_SIZE + 1}, newBody: noBody, removeErr: errRemove},
			wantErr:         domain.ErrFileIsTooLarge,
			wantCode:        codes.InvalidArgument,
			wantStatCalls:   pathOnce,
			wantRemoveCalls: pathOnce,
		},
		{
			name:               "get object not found",
			setup:              confirmSetup{statInfo: statOK, getObjectErr: domain.ErrObjectNotFound, newBody: noBody},
			wantErr:            domain.ErrFileNotUploaded,
			wantCode:           codes.FailedPrecondition,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
		},
		{
			name:               "get object failure",
			setup:              confirmSetup{statInfo: statOK, getObjectErr: errGet, newBody: noBody},
			wantErr:            errGet,
			wantCode:           codes.Unknown,
			wantNotErr:         domain.ErrFileNotUploaded,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
		},
		{
			name: "read error after five bytes",
			setup: confirmSetup{statInfo: statOK, newBody: func() *fakeBody {
				return &fakeBody{data: []byte(TEST_BODY_PDF), failAfter: 5, readErr: errRead}
			}},
			wantErr:            errRead,
			wantCode:           codes.Unknown,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantClosed:         true,
		},
		{
			name:               "empty body",
			setup:              confirmSetup{statInfo: statOK, newBody: bodyOf([]byte{})},
			wantErr:            domain.ErrFileIsEmpty,
			wantCode:           codes.InvalidArgument,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantClosed:         true,
		},
		{
			name:               "empty body, remove fails",
			setup:              confirmSetup{statInfo: statOK, newBody: bodyOf([]byte{}), removeErr: errRemove},
			wantErr:            domain.ErrFileIsEmpty,
			wantCode:           codes.InvalidArgument,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantClosed:         true,
		},
		{
			name:               "body max plus one",
			setup:              confirmSetup{statInfo: statOK, newBody: func() *fakeBody { return &fakeBody{data: paddedPDF(domain.MAX_FILE_SIZE + 1)} }},
			wantErr:            domain.ErrFileIsTooLarge,
			wantCode:           codes.InvalidArgument,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantClosed:         true,
		},
		{
			name:               "body max plus one, remove fails",
			setup:              confirmSetup{statInfo: statOK, newBody: func() *fakeBody { return &fakeBody{data: paddedPDF(domain.MAX_FILE_SIZE + 1)} }, removeErr: errRemove},
			wantErr:            domain.ErrFileIsTooLarge,
			wantCode:           codes.InvalidArgument,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantClosed:         true,
		},
		{
			name:               "text body max plus one is too large before type check",
			setup:              confirmSetup{statInfo: statOK, newBody: func() *fakeBody { return &fakeBody{data: bytes.Repeat([]byte("a"), domain.MAX_FILE_SIZE+1)} }},
			wantErr:            domain.ErrFileIsTooLarge,
			wantCode:           codes.InvalidArgument,
			wantNotErr:         domain.ErrContentTypeUnsupported,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantClosed:         true,
		},
		{
			// один байт "%" - это text/plain, а не пустой файл
			name:               "single byte body is not empty",
			setup:              confirmSetup{statInfo: statOK, newBody: bodyOf([]byte("%"))},
			wantErr:            domain.ErrContentTypeUnsupported,
			wantCode:           codes.InvalidArgument,
			wantNotErr:         domain.ErrFileIsEmpty,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantClosed:         true,
		},
		{
			name:               "text body",
			setup:              confirmSetup{statInfo: models.ObjectInfo{Size: 11}, newBody: bodyOf([]byte(TEST_BODY_TEXT))},
			wantErr:            domain.ErrContentTypeUnsupported,
			wantCode:           codes.InvalidArgument,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantClosed:         true,
		},
		{
			name:               "text body, remove fails",
			setup:              confirmSetup{statInfo: models.ObjectInfo{Size: 11}, newBody: bodyOf([]byte(TEST_BODY_TEXT)), removeErr: errRemove},
			wantErr:            domain.ErrContentTypeUnsupported,
			wantCode:           codes.InvalidArgument,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantClosed:         true,
		},
		{
			name:               "gif body",
			setup:              confirmSetup{statInfo: statOK, newBody: bodyOf([]byte(TEST_BODY_GIF))},
			wantErr:            domain.ErrContentTypeUnsupported,
			wantCode:           codes.InvalidArgument,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantClosed:         true,
		},
		{
			name:               "html body",
			setup:              confirmSetup{statInfo: statOK, newBody: bodyOf([]byte(TEST_BODY_HTML))},
			wantErr:            domain.ErrContentTypeUnsupported,
			wantCode:           codes.InvalidArgument,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantClosed:         true,
		},
		{
			name:               "declared pdf does not save text body",
			setup:              confirmSetup{statInfo: models.ObjectInfo{Size: 11, ContentType: "application/pdf"}, newBody: bodyOf([]byte(TEST_BODY_TEXT))},
			wantErr:            domain.ErrContentTypeUnsupported,
			wantCode:           codes.InvalidArgument,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantClosed:         true,
		},
		{
			name:               "file type already confirmed",
			setup:              confirmSetup{statInfo: models.ObjectInfo{Size: 15}, newBody: bodyOf([]byte(TEST_BODY_PDF)), confirmErr: domain.ErrFileTypeConfirmed},
			wantErr:            domain.ErrFileTypeConfirmed,
			wantCode:           codes.Unknown,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantConfirmCalls:   pdfConfirm,
			wantClosed:         true,
		},
		{
			name:               "file type already confirmed, remove fails",
			setup:              confirmSetup{statInfo: models.ObjectInfo{Size: 15}, newBody: bodyOf([]byte(TEST_BODY_PDF)), confirmErr: domain.ErrFileTypeConfirmed, removeErr: errRemove},
			wantErr:            domain.ErrFileTypeConfirmed,
			wantCode:           codes.Unknown,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantRemoveCalls:    pathOnce,
			wantConfirmCalls:   pdfConfirm,
			wantClosed:         true,
		},
		{
			name:               "confirm metadata not found",
			setup:              confirmSetup{statInfo: models.ObjectInfo{Size: 15}, newBody: bodyOf([]byte(TEST_BODY_PDF)), confirmErr: domain.ErrMetadataNotFound},
			wantErr:            domain.ErrMetadataNotFound,
			wantCode:           codes.Unknown,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantConfirmCalls:   pdfConfirm,
			wantClosed:         true,
		},
		{
			name:               "confirm metadata failure",
			setup:              confirmSetup{statInfo: models.ObjectInfo{Size: 15}, newBody: bodyOf([]byte(TEST_BODY_PDF)), confirmErr: errDB},
			wantErr:            errDB,
			wantCode:           codes.Unknown,
			wantStatCalls:      pathOnce,
			wantGetObjectCalls: pathOnce,
			wantConfirmCalls:   pdfConfirm,
			wantClosed:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConfirmFixture(tt.setup)

			got, err := f.svc.ConfirmUpload(t.Context(), f.userID, f.fileID)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.NotErrorIs(t, err, tt.wantNotErr)
			// ошибка удаления не подменяет причину отказа
			assert.NotErrorIs(t, err, errRemove)
			assert.Nil(t, got)
			assert.Equal(t, []getMetadataCall{{FileID: fileID, UserID: userID}}, f.repo.getCalls)
			assert.Equal(t, tt.wantStatCalls, f.storage.statCalls)
			assert.Equal(t, tt.wantGetObjectCalls, f.storage.getObjectCalls)
			assert.Equal(t, tt.wantRemoveCalls, f.storage.removeCalls)
			assert.Equal(t, tt.wantConfirmCalls, f.repo.confirmCalls)
			assert.Equal(t, tt.wantClosed, f.body.closed)
			assertNoUnrelatedCalls(t, f.repo, f.storage)
		})
	}
}

func TestConfirmUpload_ReadErrorIsNotDomainStatus(t *testing.T) {
	errRead := errors.New("read failure")

	tests := []struct {
		name      string
		failAfter int
	}{
		{name: "error after five bytes", failAfter: 5},
		{name: "error on first read", failAfter: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConfirmFixture(confirmSetup{
				statInfo: models.ObjectInfo{Size: 100},
				newBody: func() *fakeBody {
					return &fakeBody{data: []byte(TEST_BODY_PDF), failAfter: tt.failAfter, readErr: errRead}
				},
			})

			got, err := f.svc.ConfirmUpload(t.Context(), f.userID, f.fileID)

			require.ErrorIs(t, err, errRead)
			assert.Nil(t, got)
			assert.NotErrorIs(t, err, domain.ErrFileNotUploaded)
			assert.NotErrorIs(t, err, domain.ErrFileIsEmpty)
			assert.NotErrorIs(t, err, domain.ErrFileIsTooLarge)
			assert.NotErrorIs(t, err, domain.ErrContentTypeUnsupported)
			_, isStatus := status.FromError(err)
			assert.False(t, isStatus)
			assert.Nil(t, f.storage.removeCalls)
			assert.Nil(t, f.repo.confirmCalls)
			assert.True(t, f.body.closed)
		})
	}
}

func TestConfirmUpload_InfiniteBodyIsBounded(t *testing.T) {
	f := newConfirmFixture(confirmSetup{
		statInfo: models.ObjectInfo{Size: 100},
		newBody:  func() *fakeBody { return &fakeBody{infinite: true} },
	})

	got, err := f.svc.ConfirmUpload(t.Context(), f.userID, f.fileID)

	require.ErrorIs(t, err, domain.ErrFileIsTooLarge)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Nil(t, got)
	assert.LessOrEqual(t, f.body.readBytes, int64(domain.MAX_FILE_SIZE+1))
	assert.Equal(t, []string{TEST_OBJECT_PATH}, f.storage.removeCalls)
	assert.Nil(t, f.repo.confirmCalls)
	assert.True(t, f.body.closed)
}

func TestConfirmUpload_SizeAndHashMatchReadBytesInvariant(t *testing.T) {
	tests := []struct {
		name     string
		statSize int64
		body     []byte
	}{
		{name: "pdf, stat smaller", statSize: 1, body: []byte(TEST_BODY_PDF)},
		{name: "png, stat larger", statSize: 100, body: []byte(TEST_BODY_PNG)},
		{name: "jpeg, stat much larger", statSize: 1000, body: []byte(TEST_BODY_JPEG)},
		{name: "pdf exactly max", statSize: domain.MAX_FILE_SIZE, body: paddedPDF(domain.MAX_FILE_SIZE)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConfirmFixture(confirmSetup{
				statInfo: models.ObjectInfo{Size: tt.statSize},
				newBody:  bodyOf(tt.body),
			})

			_, err := f.svc.ConfirmUpload(t.Context(), f.userID, f.fileID)

			require.NoError(t, err)
			require.Len(t, f.repo.confirmCalls, 1)
			assert.Equal(t, int64(len(tt.body)), f.repo.confirmCalls[0].SizeBytes)
			assert.Equal(t, int64(len(tt.body)), f.body.readBytes)
			assert.Equal(t, sha256Hex(tt.body), f.repo.confirmCalls[0].SHA256)
		})
	}
}

// confirmInvariantCase - вход для инвариантов удаления, ConfirmMetadata и закрытия тела
type confirmInvariantCase struct {
	name             string
	setup            confirmSetup
	wantRemoveCalls  []string
	wantConfirmCount int
	wantClosed       bool
}

func confirmInvariantCases() []confirmInvariantCase {
	errDB := errors.New("db failure")
	errGet := errors.New("get object failure")
	errRead := errors.New("read failure")
	pathOnce := []string{TEST_OBJECT_PATH}
	statOK := models.ObjectInfo{Size: 100}
	pdf := bodyOf([]byte(TEST_BODY_PDF))

	return []confirmInvariantCase{
		{name: "stat zero", setup: confirmSetup{statInfo: models.ObjectInfo{Size: 0}, newBody: pdf}, wantRemoveCalls: pathOnce},
		{name: "stat max plus one", setup: confirmSetup{statInfo: models.ObjectInfo{Size: domain.MAX_FILE_SIZE + 1}, newBody: pdf}, wantRemoveCalls: pathOnce},
		{name: "empty body", setup: confirmSetup{statInfo: statOK, newBody: bodyOf([]byte{})}, wantRemoveCalls: pathOnce, wantClosed: true},
		{name: "body max plus one", setup: confirmSetup{statInfo: statOK, newBody: func() *fakeBody { return &fakeBody{data: paddedPDF(domain.MAX_FILE_SIZE + 1)} }}, wantRemoveCalls: pathOnce, wantClosed: true},
		{name: "infinite body", setup: confirmSetup{statInfo: statOK, newBody: func() *fakeBody { return &fakeBody{infinite: true} }}, wantRemoveCalls: pathOnce, wantClosed: true},
		{name: "text body", setup: confirmSetup{statInfo: statOK, newBody: bodyOf([]byte(TEST_BODY_TEXT))}, wantRemoveCalls: pathOnce, wantClosed: true},
		{name: "file type already confirmed", setup: confirmSetup{statInfo: statOK, newBody: pdf, confirmErr: domain.ErrFileTypeConfirmed}, wantRemoveCalls: pathOnce, wantConfirmCount: 1, wantClosed: true},
		{name: "success", setup: confirmSetup{statInfo: statOK, newBody: pdf}, wantConfirmCount: 1, wantClosed: true},
		{name: "metadata not found", setup: confirmSetup{getErr: domain.ErrMetadataNotFound, newBody: pdf}},
		{name: "stat object not found", setup: confirmSetup{statErr: domain.ErrObjectNotFound, newBody: pdf}},
		{name: "get object failure", setup: confirmSetup{statInfo: statOK, getObjectErr: errGet, newBody: pdf}},
		{name: "read error", setup: confirmSetup{statInfo: statOK, newBody: func() *fakeBody {
			return &fakeBody{data: []byte(TEST_BODY_PDF), failAfter: 5, readErr: errRead}
		}}, wantClosed: true},
		{name: "confirm metadata not found", setup: confirmSetup{statInfo: statOK, newBody: pdf, confirmErr: domain.ErrMetadataNotFound}, wantConfirmCount: 1, wantClosed: true},
		{name: "confirm metadata failure", setup: confirmSetup{statInfo: statOK, newBody: pdf, confirmErr: errDB}, wantConfirmCount: 1, wantClosed: true},
	}
}

func TestConfirmUpload_RemoveOnlyOnRefusalInvariant(t *testing.T) {
	for _, tt := range confirmInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			f := newConfirmFixture(tt.setup)

			_, err := f.svc.ConfirmUpload(t.Context(), f.userID, f.fileID)

			// RemoveObject вызван ровно тогда, когда ответ - один из четырёх отказов
			refused := errors.Is(err, domain.ErrFileIsEmpty) || errors.Is(err, domain.ErrFileIsTooLarge) ||
				errors.Is(err, domain.ErrContentTypeUnsupported) || errors.Is(err, domain.ErrFileTypeConfirmed)
			assert.Equal(t, tt.wantRemoveCalls != nil, refused)
			assert.Equal(t, tt.wantRemoveCalls, f.storage.removeCalls)
		})
	}
}

func TestConfirmUpload_ConfirmAtMostOnceAfterChecksInvariant(t *testing.T) {
	for _, tt := range confirmInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			f := newConfirmFixture(tt.setup)

			_, err := f.svc.ConfirmUpload(t.Context(), f.userID, f.fileID)

			assert.NotErrorIs(t, err, overreadError{})
			assert.Len(t, f.repo.confirmCalls, tt.wantConfirmCount)
		})
	}
}

func TestConfirmUpload_BodyAlwaysClosedInvariant(t *testing.T) {
	for _, tt := range confirmInvariantCases() {
		t.Run(tt.name, func(t *testing.T) {
			f := newConfirmFixture(tt.setup)

			_, err := f.svc.ConfirmUpload(t.Context(), f.userID, f.fileID)

			assert.NotErrorIs(t, err, overreadError{})
			assert.Equal(t, tt.wantClosed, f.body.closed)
		})
	}
}
