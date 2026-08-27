package memory

import (
	"context"
	"strconv"
	"testing"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/notification"
)

func TestSaveSkipsDuplicateEvent(t *testing.T) {
	r := NewNotificationRepo()
	ctx := context.Background()

	n := notification.Notification{ID: "1", EventID: "e1", Status: notification.StatusPending}

	saved, err := r.Save(ctx, n)
	if err != nil || !saved {
		t.Fatalf("первое сохранение: saved=%v err=%v", saved, err)
	}

	// Kafka доставляет "хотя бы один раз": то же событие приезжает снова.
	n.ID = "2"

	saved, err = r.Save(ctx, n)
	if err != nil {
		t.Fatal(err)
	}
	if saved {
		t.Fatal("повтор события сохранён второй раз")
	}

	_, total, err := r.List(ctx, notification.Filter{}, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("в хранилище %d записей, ожидалась 1", total)
	}
}

func TestSaveEvictsOldestAndForgetsItsEventID(t *testing.T) {
	r := NewNotificationRepo()
	r.capacity = 3
	ctx := context.Background()

	for i := range 4 {
		id := strconv.Itoa(i)
		if _, err := r.Save(ctx, notification.Notification{ID: id, EventID: "e" + id}); err != nil {
			t.Fatal(err)
		}
	}

	items, total, err := r.List(ctx, notification.Filter{}, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("в хранилище %d записей, ожидалось 3", total)
	}

	// От новых к старым: последним сохранён "3".
	if items[0].ID != "3" {
		t.Fatalf("первым отдан %q, ожидался %q", items[0].ID, "3")
	}

	// Вытесненный EventID не должен оставаться в seen - иначе map растёт
	// без границы и старое событие больше никогда не пройдёт.
	if _, ok := r.seen["e0"]; ok {
		t.Fatal("EventID вытесненной записи остался в seen")
	}
}

func TestListFiltersAndPaginates(t *testing.T) {
	r := NewNotificationRepo()
	ctx := context.Background()

	for i := range 5 {
		id := strconv.Itoa(i)

		st := notification.StatusPending
		if i%2 == 0 {
			st = notification.StatusSent
		}

		if _, err := r.Save(ctx, notification.Notification{
			ID: id, EventID: "e" + id, CustomerID: "c1", Status: st,
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Чужой клиент - пустая выборка.
	_, total, err := r.List(ctx, notification.Filter{CustomerID: "c2"}, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatalf("для чужого клиента найдено %d записей", total)
	}

	// Фильтр по статусу: sent у чётных - это 0, 2, 4.
	items, total, err := r.List(ctx, notification.Filter{Status: notification.StatusSent}, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(items) != 3 {
		t.Fatalf("sent: total=%d len=%d, ожидалось 3 и 3", total, len(items))
	}

	// Страница со смещением: total считается по всей выборке, а не по странице.
	items, total, err = r.List(ctx, notification.Filter{}, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Fatalf("total=%d, ожидалось 5", total)
	}
	if len(items) != 2 {
		t.Fatalf("на странице %d записей, ожидалось 2", len(items))
	}

	// Смещение за пределами выборки - пустая страница, а не паника.
	items, total, err = r.List(ctx, notification.Filter{}, 99, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 || total != 5 {
		t.Fatalf("за пределами: len=%d total=%d", len(items), total)
	}
}
