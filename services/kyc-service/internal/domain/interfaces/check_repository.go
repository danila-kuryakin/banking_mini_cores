package interfaces

import (
	"context"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

type CheckRepository interface {
	// SaveCheck перезаписывает результат проверки того же типа: воркер может взять
	// заявку повторно после падения, и второй строки записи в таблицу от этого быть не должно.
	SaveCheck(ctx context.Context, check models.Check) (*models.Check, error)
}
