package dto

import "time"

// Account - счёт клиента; статус отдаётся доменным именем в нижнем регистре.
type Account struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Number          string    `json:"number" example:"40817810500000000001"`
	Currency        string    `json:"currency" example:"RUB"`
	Status          string    `json:"status" example:"active" enums:"active,blocked,closed"`
	StatusChangedAt time.Time `json:"status_changed_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type OpenAccount struct {
	Currency string `json:"currency" example:"RUB"`
}

type ListAccountsQuery struct {
	Limit  int32 `form:"limit"`
	Offset int32 `form:"offset"`
}

type ListAccounts struct {
	Accounts []Account `json:"accounts"`
}

type BlockAccount struct {
	Reason string `json:"reason" example:"подозрительные операции"`
}

type UnblockAccount struct {
	Comment string `json:"comment" example:"проверка завершена"`
}

// Balance - остаток в минорных единицах валюты (копейки, центы).
type Balance struct {
	AccountID string `json:"account_id"`
	Amount    int64  `json:"amount" example:"150000"`
	Currency  string `json:"currency" example:"RUB"`
}

// StatusChange - запись истории статусов счёта; пустой from_status - открытие счёта.
type StatusChange struct {
	ID         int64     `json:"id" example:"1"`
	AccountID  string    `json:"account_id"`
	FromStatus string    `json:"from_status,omitempty" example:"active" enums:"active,blocked,closed"`
	ToStatus   string    `json:"to_status" example:"blocked" enums:"active,blocked,closed"`
	ActorID    string    `json:"actor_id,omitempty"`
	Reason     string    `json:"reason,omitempty" example:"подозрительные операции"`
	CreatedAt  time.Time `json:"created_at"`
}

type ListStatusHistoryQuery struct {
	Limit  int32 `form:"limit"`
	Offset int32 `form:"offset"`
}

type ListStatusHistory struct {
	Changes []StatusChange `json:"changes"`
}
