package dto

import "time"

// Transaction - перевод или пополнение; суммы в минорных единицах валюты.
type Transaction struct {
	ID            string    `json:"id"`
	Type          string    `json:"type" example:"transfer" enums:"transfer,deposit"`
	Status        string    `json:"status" example:"completed" enums:"completed,declined"`
	FromAccountID string    `json:"from_account_id"`
	ToAccountID   string    `json:"to_account_id"`
	Amount        int64     `json:"amount" example:"10000"`
	Currency      string    `json:"currency" example:"RUB"`
	InitiatorID   string    `json:"initiator_id,omitempty"`
	DeclineReason string    `json:"decline_reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	Entries       []Entry   `json:"entries,omitempty"`
}

// Entry - проводка: debit уменьшает баланс счёта, credit увеличивает.
type Entry struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Direction string `json:"direction" example:"debit" enums:"debit,credit"`
	Amount    int64  `json:"amount" example:"10000"`
	Currency  string `json:"currency" example:"RUB"`
}

type Transfer struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int64  `json:"amount" example:"10000"`
	Currency      string `json:"currency" example:"RUB"`
}

type Deposit struct {
	Amount   int64  `json:"amount" example:"100000"`
	Currency string `json:"currency" example:"RUB"`
}

type ListTransactionsQuery struct {
	Limit  int32 `form:"limit"`
	Offset int32 `form:"offset"`
}

type ListTransactions struct {
	Transactions []Transaction `json:"transactions"`
}
