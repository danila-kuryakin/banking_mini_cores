package models

import (
	"time"

	"github.com/google/uuid"
)

// TransactionType совпадает с меткой enum transaction_type в Postgres.
type TransactionType string

const (
	TYPE_TRANSFER TransactionType = "transfer"
	TYPE_DEPOSIT  TransactionType = "deposit"
)

// TransactionStatus совпадает с меткой enum transaction_status в Postgres.
type TransactionStatus string

const (
	STATUS_COMPLETED TransactionStatus = "completed"
	STATUS_DECLINED  TransactionStatus = "declined"
)

// Direction - сторона проводки: дебет уменьшает баланс счёта, кредит увеличивает.
type Direction string

const (
	DIRECTION_DEBIT  Direction = "debit"
	DIRECTION_CREDIT Direction = "credit"
)

// Transaction - перевод или пополнение; у отклонённой транзакции проводок нет.
type Transaction struct {
	ID            uuid.UUID
	Type          TransactionType
	Status        TransactionStatus
	FromAccountID uuid.UUID
	ToAccountID   uuid.UUID
	Amount        int64
	Currency      Currency
	InitiatorID   *uuid.UUID
	DeclineReason string
	CreatedAt     time.Time

	// Заполняются только при чтении одной транзакции.
	Entries []Entry
}

type Entry struct {
	ID            uuid.UUID
	TransactionID uuid.UUID
	AccountID     uuid.UUID
	Direction     Direction
	Amount        int64
	Currency      Currency
	CreatedAt     time.Time
}

// Posting - движение суммы со счёта From на счёт To одной транзакцией БД.
type Posting struct {
	Type        TransactionType
	From        uuid.UUID
	To          uuid.UUID
	Amount      int64
	Currency    Currency
	InitiatorID *uuid.UUID
}

// TransferCommand - перевод между счетами клиентов.
type TransferCommand struct {
	InitiatorID uuid.UUID
	From        uuid.UUID
	To          uuid.UUID
	Amount      int64
	Currency    string
}

// DepositCommand - пополнение счёта из кассы офицером.
type DepositCommand struct {
	AccountID uuid.UUID
	Amount    int64
	Currency  string
	OfficerID uuid.UUID
}

// Page - пагинация листинга: сколько записей взять и сколько пропустить.
type Page struct {
	Limit  int
	Offset int
}
