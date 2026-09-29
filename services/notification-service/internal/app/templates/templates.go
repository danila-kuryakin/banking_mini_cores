package templates

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

// draft - получатель и текст; пустой получатель пропускается.
type draft struct {
	userID string
	text   string
}

// Build превращает событие в уведомления; тип, который сервис не слушает, даёт пустой список, а не ошибку.
func Build(envelope models.Envelope) ([]models.Notification, error) {
	var (
		drafts []draft
		err    error
	)

	switch envelope.EventType {
	case models.EVENT_TRANSACTION_COMPLETED:
		drafts, err = transactionCompleted(envelope.Payload)
	case models.EVENT_TRANSACTION_DECLINED:
		drafts, err = transactionDeclined(envelope.Payload)
	case models.EVENT_APPLICATION_APPROVED:
		drafts, err = applicationApproved(envelope.Payload)
	case models.EVENT_APPLICATION_REJECTED:
		drafts, err = applicationRejected(envelope.Payload)
	case models.EVENT_DOCS_REQUESTED:
		drafts, err = docsRequested(envelope.Payload)
	default:
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", domain.ErrPoisonMessage, envelope.EventType, err)
	}

	notifications := make([]models.Notification, 0, len(drafts))

	for _, d := range drafts {
		if d.userID == "" {
			continue
		}

		userID, err := uuid.Parse(d.userID)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: user_id %q: %v", domain.ErrPoisonMessage, envelope.EventType, d.userID, err)
		}

		notifications = append(notifications, models.Notification{
			UserID:    userID,
			EventID:   envelope.EventID,
			EventType: envelope.EventType,
			Text:      d.text,
		})
	}

	return notifications, nil
}

func decode[T any](raw json.RawMessage) (T, error) {
	var payload T

	if err := json.Unmarshal(raw, &payload); err != nil {
		return payload, fmt.Errorf("payload: %w", err)
	}

	return payload, nil
}

// money печатает минорные единицы с двумя знаками после точки: у RUB, USD и EUR их два.
func money(amount int64, currency string) string {
	return fmt.Sprintf("%d.%02d %s", amount/100, amount%100, currency)
}
