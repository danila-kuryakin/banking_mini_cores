package repository

import (
	"context"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/notification"
)

// Notification — хранилище уведомлений.
//
// Интерфейс живёт в app, а реализация - в adapters: сегодня это память, завтра
// может появиться Postgres или Redis, и ни консьюмеру, ни gRPC-серверу это
// знать не нужно.
type Notification interface {
	// Save кладёт уведомление в хранилище. Второе значение - false, если
	// уведомление с таким EventID уже сохранено.
	Save(ctx context.Context, n notification.Notification) (bool, error)

	// List возвращает уведомления от новых к старым: offset штук пропускает,
	// не больше limit отдаёт. Второе значение - сколько всего подходит под
	// фильтр, без учёта постраничности.
	List(ctx context.Context, f notification.Filter, offset, limit int) ([]notification.Notification, int, error)
}
