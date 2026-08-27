package memory

import (
	"context"
	"sync"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/notification"
)

// defaultCapacity — сколько последних уведомлений держать в памяти.
//
// Хранилище тут не долговременное, а отладочное: смысл в том, чтобы посмотреть
// последние события глазами, а не хранить историю. Без верхней границы сервис
// с включённой шиной ел бы память ровно до OOM.
const defaultCapacity = 1000

// NotificationRepo — хранилище уведомлений в памяти.
//
// Кольцевого буфера как такового нет: список хранится от новых к старым, новое
// уведомление встаёт в начало, лишнее отрезается с конца. При тысяче элементов
// сдвиг среза дешевле возни с индексами, а читается такой код заметно проще.
type NotificationRepo struct {
	mu    sync.RWMutex
	items []notification.Notification

	// seen — идентификаторы уже сохранённых событий. Kafka доставляет
	// "хотя бы один раз", поэтому одно и то же событие приезжает повторно
	// после ребаланса группы или переподключения.
	seen map[string]struct{}

	capacity int
}

func NewNotificationRepo() *NotificationRepo {
	return &NotificationRepo{
		items:    make([]notification.Notification, 0, defaultCapacity),
		seen:     make(map[string]struct{}, defaultCapacity),
		capacity: defaultCapacity,
	}
}

// Save кладёт уведомление в начало списка. Возвращает false, если событие с
// таким EventID уже сохранено.
func (r *NotificationRepo) Save(_ context.Context, n notification.Notification) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if n.EventID != "" {
		if _, dup := r.seen[n.EventID]; dup {
			return false, nil
		}

		r.seen[n.EventID] = struct{}{}
	}

	r.items = append([]notification.Notification{n}, r.items...)

	// Вытесняя старое из списка, вычищаем и его EventID - иначе map растёт
	// без границы, и вся экономия на длине списка теряется.
	for len(r.items) > r.capacity {
		evicted := r.items[len(r.items)-1]
		r.items = r.items[:len(r.items)-1]
		delete(r.seen, evicted.EventID)
	}

	return true, nil
}

// List отдаёт страницу уведомлений от новых к старым.
func (r *NotificationRepo) List(_ context.Context, f notification.Filter, offset, limit int) ([]notification.Notification, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	matched := make([]notification.Notification, 0, len(r.items))

	for _, n := range r.items {
		if f.Matches(n) {
			matched = append(matched, n)
		}
	}

	total := len(matched)

	if offset >= total {
		return nil, total, nil
	}

	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}

	// Копия, а не срез поверх matched: matched уже собран заново на каждый
	// вызов, но вернуть кусок под RLock и дать вызывающему писать в него -
	// приглашение к гонке при следующей правке этого кода.
	page := make([]notification.Notification, end-offset)
	copy(page, matched[offset:end])

	return page, total, nil
}
