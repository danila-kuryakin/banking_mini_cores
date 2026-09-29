package domain

import (
	"time"
)

const (
	DEFAULT_REQUEST_TIMEOUT = 30 * time.Second
)

const (
	DEFAULT_PAGE_SIZE = 20
	MAX_PAGE_SIZE     = 100
	// FILE_LIST_LIMIT - сколько файлов клиента забирать у filestore, чтобы
	// собрать набор подтверждённых типов. Типов всего три, столько файлов с
	// запасом хватает.
	FILE_LIST_LIMIT = 100
)

// Причины переходов, которые kyc записывает клиенту в customer-service.
const (
	REASON_APPLICATION_SUBMITTED = "kyc application submitted"
	REASON_APPLICATION_APPROVED  = "kyc application approved"
)

// Авто-проверки.
const (
	MIN_CUSTOMER_AGE_YEARS = 18

	DEFAULT_AUTO_CHECKS_INTERVAL   = 5 * time.Second
	DEFAULT_AUTO_CHECKS_BATCH_SIZE = 20
)

// REASON_AUTO_REJECTED - причина возврата клиента в profile_filled после
// авто-отказа. К ней дописывается, какая именно проверка не прошла.
const REASON_AUTO_REJECTED = "kyc auto check failed"

// Публикация outbox в Kafka.
const (
	DEFAULT_OUTBOX_INTERVAL   = time.Second
	DEFAULT_OUTBOX_BATCH_SIZE = 100
)

const COMMENT_MAX_LENGTH = 500

// Ограничения записи санкционного списка.
const (
	NAME_MAX_LENGTH   = 100
	SOURCE_MAX_LENGTH = 200
)
