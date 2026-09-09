package domain

import (
	"time"
)

const (
	DEFAULT_REQUEST_TIMEOUT = 30 * time.Second
)

const (
	DEFAULT_MINIO_CONNECT_TIMEOUT   = 5 * time.Second
	DEFAULT_MINIO_HEALTH_CHECK_FREQ = 15 * time.Second
)

const (
	NO_SUCH_KEY_CODE = "NoSuchKey"
)

const (
	DEFAULT_PAGE_SIZE = 20
	MAX_PAGE_SIZE     = 100
)

const (
	MAX_FILE_SIZE        = 10 << 20 // 10 МБ - лимит из ТЗ
	CONTENT_SNIFF_LENGTH = 512      // http.DetectContentType дальше первых 512 байт не смотрит
	FILENAME_MAX_LENGTH  = 255
)

// Сроки жизни presigned-ссылок.
const (
	UPLOAD_URL_TTL   = 15 * time.Minute
	DOWNLOAD_URL_TTL = 5 * time.Minute
)

// Формат пути: {user_id}/{type}/{file_id}.
const OBJECT_PATH_FORMAT = "%s/%s/%s"
