package models

import (
	"time"

	"github.com/google/uuid"
)

// Account - счёт клиента. Баланса здесь нет: источник истины по деньгам - ledger-service.
type Account struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Number          string
	Currency        Currency
	Status          AccountStatus
	StatusChangedAt time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Transition - смена статуса счёта. Пустой ActorID - системный переход.
type Transition struct {
	To      AccountStatus
	ActorID *uuid.UUID
	Reason  string
}

// StatusChange - запись истории статусов; пустой From - запись об открытии счёта.
type StatusChange struct {
	ID        int64
	AccountID uuid.UUID
	From      AccountStatus
	To        AccountStatus
	ActorID   *uuid.UUID
	Reason    string
	CreatedAt time.Time
}

// Page - пагинация листинга: сколько записей взять и сколько пропустить.
type Page struct {
	Limit  int
	Offset int
}

// Balance - остаток счёта в минорных единицах.
type Balance struct {
	AccountID uuid.UUID
	Amount    int64
	Currency  Currency
}
