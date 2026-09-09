package dto

import (
	"time"
)

// InitUpload - заявка на загрузку. Сам файл сюда не приходит: в ответ выдаётся
// ссылка, по которой клиент кладёт файл в хранилище напрямую.
type InitUpload struct {
	Type     string `json:"type" example:"passport" enums:"passport,selfie,proof_of_address"`
	Filename string `json:"filename" example:"passport.png"`
}

type UploadTicket struct {
	FileID    string    `json:"file_id"`
	UploadURL string    `json:"upload_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type DownloadTicket struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type File struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Type        string     `json:"type" example:"passport"`
	Status      string     `json:"status" example:"confirmed"`
	Filename    string     `json:"filename"`
	ContentType string     `json:"content_type" example:"image/png"`
	SizeBytes   int64      `json:"size_bytes"`
	SHA256      string     `json:"sha256"`
	CreatedAt   time.Time  `json:"created_at"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
}

type ListFilesQuery struct {
	Limit  int32 `form:"limit"`
	Offset int32 `form:"offset"`
}

type ListFiles struct {
	Files []File `json:"files"`
}

type RequiredFiles struct {
	OK      bool     `json:"ok"`
	Missing []string `json:"missing" example:"selfie"`
}
