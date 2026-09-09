package models

import (
	"time"

	"github.com/google/uuid"
)

type FileType string

const (
	FILE_TYPE_PASSPORT         FileType = "passport"
	FILE_TYPE_SELFIE           FileType = "selfie"
	FILE_TYPE_PROOF_OF_ADDRESS FileType = "proof_of_address"
)

type FileStatus string

const (
	FILE_STATUS_UPLOADED  FileStatus = "uploaded"
	FILE_STATUS_CONFIRMED FileStatus = "confirmed"
)

var REQUIRED_FILE_TYPES = []FileType{
	FILE_TYPE_PASSPORT,
	FILE_TYPE_SELFIE,
}

var ALLOWED_CONTENT_TYPES = map[string]struct{}{
	"image/jpeg":      {},
	"image/png":       {},
	"application/pdf": {},
}

func IsAllowedContentType(contentType string) bool {
	_, ok := ALLOWED_CONTENT_TYPES[contentType]

	return ok
}

type Metadata struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Type        FileType
	Status      FileStatus
	ObjectPath  string
	Filename    string
	ContentType string
	SizeBytes   int64
	SHA256      string
	CreatedAt   time.Time
	ConfirmedAt *time.Time
}

// PresignedURL - ссылка на прямой обмен с хранилищем и её срок годности.
type PresignedURL struct {
	URL       string
	ExpiresAt time.Time
}

// DownloadOptions - как хранилище должно представить файл при скачивании.
type DownloadOptions struct {
	Filename    string // исходное имя у клиента; пустое - заголовок не ставится
	ContentType string // реальный тип, определённый на ConfirmUpload
}
