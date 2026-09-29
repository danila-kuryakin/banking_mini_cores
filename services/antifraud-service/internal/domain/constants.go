package domain

import "time"

const (
	DEFAULT_REQUEST_TIMEOUT = 30 * time.Second
)

// Коды правил - они же ключи строк af_rules.
const (
	RULE_SINGLE_AMOUNT_LIMIT = "single_amount_limit"
	RULE_VELOCITY            = "velocity"
	RULE_DAILY_AMOUNT_LIMIT  = "daily_amount_limit"
	RULE_SAME_ACCOUNT        = "same_account"
)

// Паузы между повторами обработки события из Kafka: удваиваются от MIN до MAX.
const (
	DEFAULT_RETRY_MIN_BACKOFF = 500 * time.Millisecond
	DEFAULT_RETRY_MAX_BACKOFF = 30 * time.Second
)
