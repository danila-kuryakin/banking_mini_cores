package domain

import "time"

const (
	DEFAULT_REQUEST_TIMEOUT = 30 * time.Second
)

const (
	DEFAULT_PAGE_SIZE = 20
	MAX_PAGE_SIZE     = 100
)

// Повторы обработки события: паузы удваиваются от MIN до MAX, после MAX_ATTEMPTS событие уходит в DLQ.
const (
	DEFAULT_RETRY_MAX_ATTEMPTS = 5
	DEFAULT_RETRY_MIN_BACKOFF  = 500 * time.Millisecond
	DEFAULT_RETRY_MAX_BACKOFF  = 30 * time.Second
)

// Заголовки, которые сообщение получает в DLQ: откуда оно и почему не обработано.
const (
	DLQ_HEADER_SOURCE_TOPIC = "dlq_source_topic"
	DLQ_HEADER_PARTITION    = "dlq_source_partition"
	DLQ_HEADER_OFFSET       = "dlq_source_offset"
	DLQ_HEADER_ERROR        = "dlq_error"
	DLQ_HEADER_FAILED_AT    = "dlq_failed_at"
)
