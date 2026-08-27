package kafka

import (
	"context"
	"time"
)

// Типы событий, которые публикует kyc-service. Схема имени -
// <сервис>.<сущность>.<что произошло>, в прошедшем времени: событие сообщает
// о свершившемся факте, а не просит что-то сделать.
const (
	EventApplicationSubmitted = "kyc.application.submitted"
	EventApplicationApproved  = "kyc.application.approved"
	EventApplicationRejected  = "kyc.application.rejected"
)

// ApplicationSubmitted — клиент подал анкету на проверку.
//
// Документов и их содержимого тут нет: KYC-анкета целиком состоит из
// персональных данных, а событие живёт в топике до истечения retention.
// Подписчику достаточно идентификаторов, подробности он спросит по gRPC.
type ApplicationSubmitted struct {
	ApplicationID string    `json:"application_id"`
	CustomerID    string    `json:"customer_id"`
	SubmittedAt   time.Time `json:"submitted_at"`
}

// ApplicationApproved — анкета одобрена. По этому событию customer-service
// переводит клиента в активный статус, а account-service может открывать счёт.
type ApplicationApproved struct {
	ApplicationID string    `json:"application_id"`
	CustomerID    string    `json:"customer_id"`
	OfficerID     string    `json:"officer_id,omitempty"`
	ApprovedAt    time.Time `json:"approved_at"`
}

// ApplicationRejected — анкета отклонена.
//
// Reason - это машинный код причины, а не текст для клиента: по нему подписчик
// принимает решение, а формулировку показывает уже своя витрина.
type ApplicationRejected struct {
	ApplicationID string    `json:"application_id"`
	CustomerID    string    `json:"customer_id"`
	OfficerID     string    `json:"officer_id,omitempty"`
	Reason        string    `json:"reason"`
	RejectedAt    time.Time `json:"rejected_at"`
}

// PublishApplicationSubmitted публикует событие о подаче анкеты.
//
// Ключом во всех событиях берётся CustomerID, а не ApplicationID: анкет у
// клиента может быть несколько (отказ, повторная подача), и подписчику важно
// видеть их в том порядке, в каком они происходили.
func (p *Producer) PublishApplicationSubmitted(ctx context.Context, ev ApplicationSubmitted) error {
	return p.publish(ctx, EventApplicationSubmitted, ev.CustomerID, ev)
}

// PublishApplicationApproved публикует событие об одобрении анкеты.
func (p *Producer) PublishApplicationApproved(ctx context.Context, ev ApplicationApproved) error {
	return p.publish(ctx, EventApplicationApproved, ev.CustomerID, ev)
}

// PublishApplicationRejected публикует событие об отказе по анкете.
func (p *Producer) PublishApplicationRejected(ctx context.Context, ev ApplicationRejected) error {
	return p.publish(ctx, EventApplicationRejected, ev.CustomerID, ev)
}
