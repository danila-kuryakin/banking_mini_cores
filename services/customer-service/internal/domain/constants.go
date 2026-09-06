package domain

import (
	"time"
)

const (
	DEFAULT_PAGE_SIZE = 20
	MAX_PAGE_SIZE     = 100
)

const (
	NAME_MAX_LENGTH        = 100
	CITIZENSHIP_MAX_LENGTH = 64
	PHONE_PATTERN          = `^\+?[0-9]{7,15}$`
	MAX_CUSTOMER_AGE_YEARS = 120
)

const (
	DEFAULT_REQUEST_TIMEOUT = 30 * time.Second
)
