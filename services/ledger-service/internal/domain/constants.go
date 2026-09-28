package domain

import "time"

const (
	DEFAULT_REQUEST_TIMEOUT = 30 * time.Second
)

const (
	DEFAULT_PAGE_SIZE = 20
	MAX_PAGE_SIZE     = 100
)

// Публикация outbox в Kafka.
const (
	DEFAULT_OUTBOX_INTERVAL   = time.Second
	DEFAULT_OUTBOX_BATCH_SIZE = 100
)

// Причины отказа, которые пишутся в transactions.decline_reason.
const (
	DECLINE_INSUFFICIENT_FUNDS = "insufficient funds"
	DECLINE_ANTIFRAUD          = "denied by antifraud"
)
