package domain

import (
	"time"
)

//const (
//	ROLE_CLIENT  = "client"
//	ROLE_OFFICER = "officer"
//	ROLE_ADMIN   = "admin"
//)

const (
	DEFAULT_PAGE_SIZE = 20
	MAX_PAGE_SIZE     = 100
)

const (
	NAME_MAX_LENGTH        = 100
	CITIZENSHIP_MAX_LENGTH = 64
	PHONE_PATTERN          = `^\+?[0-9]{7,15}$`
	PAGE_TOKEN_MAX_LENGTH  = 512
	MAX_CUSTOMER_AGE_YEARS = 120
)

const (
	DEFAULT_REQEST_TIMEOUT = 30 * time.Second
)
