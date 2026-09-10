package interfaces

import (
	"context"

	"github.com/google/uuid"
)

// CustomerProfiles - то, что auth-service требует от customer-service. Порт
// объявлен доменными типами, чтобы сервисный слой не зависел ни от
// сгенерированного клиента, ни от gRPC вообще, и подменялся заглушкой в тестах.
type CustomerProfiles interface {
	// CreateProfile заводит карточку клиента. Если карточка уже есть, возвращает
	// domain.ErrProfileAlreadyExists - вызывающий решает, ошибка это или нет.
	CreateProfile(ctx context.Context, userID uuid.UUID) error
}
