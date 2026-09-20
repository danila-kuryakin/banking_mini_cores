package interfaces

import (
	"context"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

type SanctionsRepository interface {
	// FindMatch возвращает nil без ошибки, если человека в списке нет: отсутствие
	// совпадения - штатный исход проверки, а не сбой.
	FindMatch(ctx context.Context, firstName, lastName string, birthDate time.Time) (*models.SanctionEntry, error)
	// Add добавляет человека в санкционный список; такой же человек уже в списке - ErrSanctionEntryExists.
	Add(ctx context.Context, entry models.SanctionEntry) (*models.SanctionEntry, error)
	// List отдаёт санкционный список постранично.
	List(ctx context.Context, limit, offset int) ([]models.SanctionEntry, error)
}
