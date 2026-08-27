package kafka

import (
	"context"
	"time"
)

// Типы событий, которые публикует ledger-service. Схема имени -
// <сервис>.<сущность>.<что произошло>, в прошедшем времени: событие сообщает
// о свершившемся факте, а не просит что-то сделать.
const (
	EventTransactionPosted = "ledger.transaction.posted"
	EventTransactionFailed = "ledger.transaction.failed"
)

// Money повторяет common.v1.Money: сумма в минорных единицах (копейках) и
// код валюты.
//
// Целое число, а не float: деньги в плавающей точке рано или поздно дают
// расхождение на копейку, и в реестре проводок это недопустимо.
type Money struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// TransactionPosted — проводка проведена и записана в реестр.
type TransactionPosted struct {
	TransactionID string    `json:"transaction_id"`
	Type          string    `json:"type"`
	Amount        Money     `json:"amount"`
	FromAccountID string    `json:"from_account_id,omitempty"`
	ToAccountID   string    `json:"to_account_id,omitempty"`
	PostedAt      time.Time `json:"posted_at"`
}

// TransactionFailed — проводка не прошла: не хватило средств, счёт заблокирован
// или её отклонил antifraud.
//
// Reason - машинный код причины, а не текст для клиента.
type TransactionFailed struct {
	TransactionID string    `json:"transaction_id"`
	Type          string    `json:"type"`
	Amount        Money     `json:"amount"`
	FromAccountID string    `json:"from_account_id,omitempty"`
	ToAccountID   string    `json:"to_account_id,omitempty"`
	Reason        string    `json:"reason"`
	FailedAt      time.Time `json:"failed_at"`
}

// PublishTransactionPosted публикует событие о проведённой проводке.
func (p *Producer) PublishTransactionPosted(ctx context.Context, ev TransactionPosted) error {
	return p.publish(ctx, EventTransactionPosted, partitionKey(ev.FromAccountID, ev.ToAccountID), ev)
}

// PublishTransactionFailed публикует событие об отклонённой проводке.
func (p *Producer) PublishTransactionFailed(ctx context.Context, ev TransactionFailed) error {
	return p.publish(ctx, EventTransactionFailed, partitionKey(ev.FromAccountID, ev.ToAccountID), ev)
}

// partitionKey выбирает ключ партиции для проводки.
//
// Ключ - счёт-источник: по нему считают баланс и лимиты, поэтому именно его
// события подписчику важно видеть строго по порядку. У пополнения источника
// нет, и тогда ключом становится счёт-получатель.
//
// Порядок гарантируется только для одной стороны проводки: перевод между
// двумя счетами попадёт в партицию отправителя, и относительно событий
// получателя он не упорядочен. Подписчику, который считает баланс обеих
// сторон, придётся смотреть на TransactionID, а не на порядок доставки.
func partitionKey(from, to string) string {
	if from != "" {
		return from
	}

	return to
}
