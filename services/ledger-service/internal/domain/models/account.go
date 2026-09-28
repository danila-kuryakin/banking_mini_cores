package models

import "github.com/google/uuid"

// AccountStatus - статус счёта в account-service, леджеру важно только active.
type AccountStatus string

const (
	ACCOUNT_STATUS_ACTIVE  AccountStatus = "active"
	ACCOUNT_STATUS_BLOCKED AccountStatus = "blocked"
	ACCOUNT_STATUS_CLOSED  AccountStatus = "closed"
)

// AccountInfo - счёт глазами леджера.
type AccountInfo struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	Currency Currency
	Status   AccountStatus
}

// Balance - остаток счёта в минорных единицах.
type Balance struct {
	AccountID uuid.UUID
	Amount    int64
	Currency  Currency
}

// Verdict - решение антифрода по переводу.
type Verdict struct {
	Allowed bool
	Reasons []string
}
