package templates

import (
	"encoding/json"
	"fmt"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

// transactionCompleted: перевод - обеим сторонам, между своими счетами - одно; пополнение - получателю.
func transactionCompleted(raw json.RawMessage) ([]draft, error) {
	tx, err := decode[models.TransactionPayload](raw)
	if err != nil {
		return nil, err
	}

	sum := money(tx.Amount, tx.Currency)

	switch tx.Type {
	case models.TRANSACTION_TYPE_DEPOSIT:
		return []draft{{
			userID: tx.ToUserID,
			text:   fmt.Sprintf("Счёт %s пополнен на %s.", tx.ToAccountID, sum),
		}}, nil
	case models.TRANSACTION_TYPE_TRANSFER:
		sender := senderOf(tx)

		if sender != "" && sender == tx.ToUserID {
			return []draft{{
				userID: sender,
				text:   fmt.Sprintf("Перевод %s между своими счетами выполнен: %s -> %s.", sum, tx.FromAccountID, tx.ToAccountID),
			}}, nil
		}

		return []draft{
			{
				userID: sender,
				text:   fmt.Sprintf("Перевод %s на счёт %s выполнен.", sum, tx.ToAccountID),
			},
			{
				userID: tx.ToUserID,
				text:   fmt.Sprintf("Поступление %s на счёт %s со счёта %s.", sum, tx.ToAccountID, tx.FromAccountID),
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown transaction type %q", tx.Type)
	}
}

// transactionDeclined: об отказе узнаёт только отправитель, получателю чужие неудачи ни к чему.
func transactionDeclined(raw json.RawMessage) ([]draft, error) {
	tx, err := decode[models.TransactionPayload](raw)
	if err != nil {
		return nil, err
	}

	return []draft{{
		userID: senderOf(tx),
		text:   fmt.Sprintf("Перевод %s на счёт %s отклонён: %s.", money(tx.Amount, tx.Currency), tx.ToAccountID, tx.DeclineReason),
	}}, nil
}

// senderOf - владелец счёта-источника; у старых событий без from_user_id им считается инициатор.
func senderOf(tx models.TransactionPayload) string {
	if tx.FromUserID != "" {
		return tx.FromUserID
	}

	return tx.InitiatorID
}
