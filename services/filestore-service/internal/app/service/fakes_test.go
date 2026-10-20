package service

import (
	"context"
	"io"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
	"github.com/google/uuid"
)

var _ interfaces.MetadataRepository = (*fakeMetadataRepo)(nil)
var _ interfaces.MinIORepository = (*fakeMinIO)(nil)
var _ io.ReadCloser = (*fakeBody)(nil)

// getMetadataCall - аргументы одного вызова GetMetadata
type getMetadataCall struct {
	FileID uuid.UUID
	UserID uuid.UUID
}

// confirmMetadataCall - аргументы одного вызова ConfirmMetadata
type confirmMetadataCall struct {
	FileID      uuid.UUID
	UserID      uuid.UUID
	ContentType string
	SizeBytes   int64
	SHA256      string
}

// listMetadataCall - аргументы одного вызова ListMetadata
type listMetadataCall struct {
	UserID uuid.UUID
	Limit  int
	Offset int
}

// fakeMetadataRepo отдаёт заданный на метод ответ и записывает вызовы
type fakeMetadataRepo struct {
	createResult *models.Metadata
	createErr    error
	createCalls  []models.Metadata

	getResult *models.Metadata
	getErr    error
	getCalls  []getMetadataCall

	confirmResult *models.Metadata
	confirmErr    error
	confirmCalls  []confirmMetadataCall

	listResult []*models.Metadata
	listErr    error
	listCalls  []listMetadataCall

	confirmedTypes     []models.FileType
	listConfirmedErr   error
	listConfirmedCalls []uuid.UUID
}

func (r *fakeMetadataRepo) CreateMetadata(_ context.Context, metadata models.Metadata) (*models.Metadata, error) {
	r.createCalls = append(r.createCalls, metadata)
	if r.createErr != nil {
		return nil, r.createErr
	}
	return r.createResult, nil
}

func (r *fakeMetadataRepo) GetMetadata(_ context.Context, fileID, userID uuid.UUID) (*models.Metadata, error) {
	r.getCalls = append(r.getCalls, getMetadataCall{FileID: fileID, UserID: userID})
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.getResult, nil
}

func (r *fakeMetadataRepo) ConfirmMetadata(_ context.Context, fileID, userID uuid.UUID, contentType string, sizeBytes int64, sha256 string) (*models.Metadata, error) {
	r.confirmCalls = append(r.confirmCalls, confirmMetadataCall{
		FileID:      fileID,
		UserID:      userID,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		SHA256:      sha256,
	})
	if r.confirmErr != nil {
		return nil, r.confirmErr
	}
	return r.confirmResult, nil
}

func (r *fakeMetadataRepo) ListMetadata(_ context.Context, userID uuid.UUID, limit, offset int) ([]*models.Metadata, error) {
	r.listCalls = append(r.listCalls, listMetadataCall{UserID: userID, Limit: limit, Offset: offset})
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.listResult, nil
}

func (r *fakeMetadataRepo) ListConfirmedTypes(_ context.Context, userID uuid.UUID) ([]models.FileType, error) {
	r.listConfirmedCalls = append(r.listConfirmedCalls, userID)
	if r.listConfirmedErr != nil {
		return nil, r.listConfirmedErr
	}
	return r.confirmedTypes, nil
}

// presignPutCall - аргументы одного вызова PresignedPutURL
type presignPutCall struct {
	ObjectPath string
	Expiry     time.Duration
}

// presignGetCall - аргументы одного вызова PresignedGetURL
type presignGetCall struct {
	ObjectPath string
	Expiry     time.Duration
	Opts       models.DownloadOptions
}

// fakeMinIO отдаёт заданный на метод ответ и записывает вызовы
type fakeMinIO struct {
	putURL   string
	putErr   error
	putCalls []presignPutCall

	getURL      string
	getURLErr   error
	getURLCalls []presignGetCall

	statInfo  models.ObjectInfo
	statErr   error
	statCalls []string

	body           *fakeBody
	getObjectErr   error
	getObjectCalls []string

	removeErr   error
	removeCalls []string
}

func (m *fakeMinIO) PresignedPutURL(_ context.Context, objectPath string, expiry time.Duration) (string, error) {
	m.putCalls = append(m.putCalls, presignPutCall{ObjectPath: objectPath, Expiry: expiry})
	if m.putErr != nil {
		return "", m.putErr
	}
	return m.putURL, nil
}

func (m *fakeMinIO) PresignedGetURL(_ context.Context, objectPath string, expiry time.Duration, opts models.DownloadOptions) (string, error) {
	m.getURLCalls = append(m.getURLCalls, presignGetCall{ObjectPath: objectPath, Expiry: expiry, Opts: opts})
	if m.getURLErr != nil {
		return "", m.getURLErr
	}
	return m.getURL, nil
}

func (m *fakeMinIO) StatObject(_ context.Context, objectPath string) (models.ObjectInfo, error) {
	m.statCalls = append(m.statCalls, objectPath)
	if m.statErr != nil {
		return models.ObjectInfo{}, m.statErr
	}
	return m.statInfo, nil
}

func (m *fakeMinIO) GetObject(_ context.Context, objectPath string) (io.ReadCloser, error) {
	m.getObjectCalls = append(m.getObjectCalls, objectPath)
	if m.getObjectErr != nil {
		return nil, m.getObjectErr
	}
	// тела нет - по контракту объекта нет
	if m.body == nil {
		return nil, domain.ErrObjectNotFound
	}
	return m.body, nil
}

func (m *fakeMinIO) RemoveObject(_ context.Context, objectPath string) error {
	m.removeCalls = append(m.removeCalls, objectPath)
	return m.removeErr
}

// overreadError - бесконечное тело прочитали сверх страховочного предела
type overreadError struct{}

func (overreadError) Error() string { return "fake body: infinite body read beyond safety cap" }

// fakeBody - тело объекта: заданные байты, ошибка чтения после N байт или бесконечный поток нулей
type fakeBody struct {
	data      []byte
	failAfter int
	readErr   error
	infinite  bool
	chunk     int

	pos        int
	readBytes  int64
	closeCalls int
	closed     bool
}

func (b *fakeBody) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := len(p)
	if b.chunk > 0 && n > b.chunk {
		n = b.chunk
	}
	if b.infinite {
		// страховка от зависания: дальше 2*(MAX+1) байт бесконечное тело не отдаёт
		if b.readBytes >= 2*(domain.MAX_FILE_SIZE+1) {
			return 0, overreadError{}
		}
		clear(p[:n])
		b.readBytes += int64(n)
		return n, nil
	}
	limit := len(b.data)
	if b.readErr != nil {
		limit = b.failAfter
	}
	remaining := limit - b.pos
	if remaining <= 0 {
		if b.readErr != nil {
			return 0, b.readErr
		}
		return 0, io.EOF
	}
	if n > remaining {
		n = remaining
	}
	copy(p, b.data[b.pos:b.pos+n])
	b.pos += n
	b.readBytes += int64(n)
	return n, nil
}

func (b *fakeBody) Close() error {
	b.closeCalls++
	b.closed = true
	return nil
}
