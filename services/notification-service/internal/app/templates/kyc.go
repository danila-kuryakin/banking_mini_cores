package templates

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

// documentNames - типы документов kyc-service по-русски; неизвестный тип печатается как есть.
var documentNames = map[string]string{
	"passport":         "паспорт",
	"selfie":           "селфи",
	"proof_of_address": "подтверждение адреса",
}

func applicationApproved(raw json.RawMessage) ([]draft, error) {
	app, err := decode[models.ApplicationPayload](raw)
	if err != nil {
		return nil, err
	}

	return []draft{{userID: app.UserID, text: "Поздравляем, вы верифицированы."}}, nil
}

// applicationRejected: авто-отказ можно исправить новой заявкой, отказ офицера - окончательный.
func applicationRejected(raw json.RawMessage) ([]draft, error) {
	app, err := decode[models.ApplicationPayload](raw)
	if err != nil {
		return nil, err
	}

	switch app.Status {
	case models.APPLICATION_STATUS_AUTO_REJECTED:
		return []draft{{
			userID: app.UserID,
			text:   fmt.Sprintf("Заявка на верификацию не прошла автоматическую проверку: %s. Исправьте анкету и подайте заявку снова.", app.Reason),
		}}, nil
	case models.APPLICATION_STATUS_REJECTED:
		return []draft{{
			userID: app.UserID,
			text:   fmt.Sprintf("Заявка на верификацию отклонена: %s.", app.Reason),
		}}, nil
	default:
		return nil, fmt.Errorf("unexpected status %q for rejection", app.Status)
	}
}

func docsRequested(raw json.RawMessage) ([]draft, error) {
	app, err := decode[models.ApplicationPayload](raw)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(app.RequestedTypes))
	for _, t := range app.RequestedTypes {
		if name, ok := documentNames[t]; ok {
			names = append(names, name)
		} else {
			names = append(names, t)
		}
	}

	text := fmt.Sprintf("Для проверки нужны документы: %s.", strings.Join(names, ", "))
	if app.Reason != "" {
		text += " Комментарий сотрудника: " + app.Reason
	}

	return []draft{{userID: app.UserID, text: text}}, nil
}
