package domain

import "time"

const (
	DEFAULT_REQUEST_TIMEOUT = 30 * time.Second
)

const (
	DEFAULT_PAGE_SIZE = 20
	MAX_PAGE_SIZE     = 100
)

// MAX_ACCOUNTS_PER_CURRENCY - лимит незакрытых счетов клиента в одной валюте.
const MAX_ACCOUNTS_PER_CURRENCY = 3

const REASON_MAX_LENGTH = 500

// ACCOUNT_NUMBER_LENGTH - длина номера счёта в цифрах.
const ACCOUNT_NUMBER_LENGTH = 20
