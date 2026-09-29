package models

// События ledger-service из топика ledger.events.
const (
	EVENT_TRANSACTION_COMPLETED = "ledger.transaction_completed"
	EVENT_TRANSACTION_DECLINED  = "ledger.transaction_declined"

	TRANSACTION_TYPE_TRANSFER = "transfer"
	TRANSACTION_TYPE_DEPOSIT  = "deposit"
)

// TransactionPayload - payload событий ledger.transaction_*; пустой user_id - владелец неизвестен.
type TransactionPayload struct {
	TransactionID string `json:"transaction_id"`
	Type          string `json:"type"`
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	InitiatorID   string `json:"initiator_id"`
	FromUserID    string `json:"from_user_id"`
	ToUserID      string `json:"to_user_id"`
	DeclineReason string `json:"decline_reason"`
}
