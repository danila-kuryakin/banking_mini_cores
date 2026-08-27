package grpc_server

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/adapters/memory"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/notification"
	commonv1 "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/pb/gen/common/v1"
	notificationv1 "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/pb/gen/notification/v1"
)

func newServer(t *testing.T, items ...notification.Notification) *NotificationServer {
	t.Helper()

	repo := memory.NewRepository()
	ctx := context.Background()

	for _, n := range items {
		if _, err := repo.Notification.Save(ctx, n); err != nil {
			t.Fatal(err)
		}
	}

	return NewNotification(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestListNotificationsEmpty(t *testing.T) {
	s := newServer(t)

	resp, err := s.ListNotifications(context.Background(), &notificationv1.ListNotificationsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.GetNotifications()) != 0 {
		t.Fatalf("вернулось %d уведомлений, ожидалось 0", len(resp.GetNotifications()))
	}
	if resp.GetPage().GetNextPageToken() != "" {
		t.Fatal("на пустой выборке выдан токен следующей страницы")
	}
}

func TestListNotificationsMapsFields(t *testing.T) {
	created := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

	s := newServer(t, notification.Notification{
		ID:         "n-1",
		CustomerID: "c-1",
		EventID:    "e-1",
		EventType:  "auth.user.registered",
		Channel:    notification.ChannelEmail,
		Status:     notification.StatusPending,
		Subject:    "Welcome aboard",
		Body:       `{"user_id":"u-1"}`,
		CreatedAt:  created,
	})

	resp, err := s.ListNotifications(context.Background(), &notificationv1.ListNotificationsRequest{})
	if err != nil {
		t.Fatal(err)
	}

	got := resp.GetNotifications()[0]

	if got.GetNotificationId() != "n-1" || got.GetEventId() != "e-1" {
		t.Fatalf("идентификаторы не совпали: %+v", got)
	}
	if got.GetChannel() != notificationv1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL {
		t.Fatalf("channel = %v", got.GetChannel())
	}
	if got.GetStatus() != notificationv1.NotificationStatus_NOTIFICATION_STATUS_PENDING {
		t.Fatalf("status = %v", got.GetStatus())
	}
	if !got.GetCreatedAt().AsTime().Equal(created) {
		t.Fatalf("created_at = %v", got.GetCreatedAt().AsTime())
	}

	// Неотправленное уведомление не должно приезжать с sent_at = 1970 год.
	if got.GetSentAt() != nil {
		t.Fatalf("sent_at заполнен у неотправленного уведомления: %v", got.GetSentAt().AsTime())
	}
}

func TestListNotificationsPaginates(t *testing.T) {
	items := make([]notification.Notification, 0, 5)
	for i := range 5 {
		id := strconv.Itoa(i)
		items = append(items, notification.Notification{ID: id, EventID: "e" + id})
	}

	s := newServer(t, items...)
	ctx := context.Background()

	resp, err := s.ListNotifications(ctx, &notificationv1.ListNotificationsRequest{
		Page: &commonv1.PageRequest{PageSize: 2},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.GetNotifications()) != 2 {
		t.Fatalf("на первой странице %d записей, ожидалось 2", len(resp.GetNotifications()))
	}
	if resp.GetPage().GetTotalSize() != 5 {
		t.Fatalf("total_size = %d, ожидалось 5", resp.GetPage().GetTotalSize())
	}
	if resp.GetPage().GetNextPageToken() != "2" {
		t.Fatalf("next_page_token = %q, ожидался %q", resp.GetPage().GetNextPageToken(), "2")
	}

	// Последняя страница: токена дальше быть не должно.
	resp, err = s.ListNotifications(ctx, &notificationv1.ListNotificationsRequest{
		Page: &commonv1.PageRequest{PageSize: 2, PageToken: "4"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetNotifications()) != 1 {
		t.Fatalf("на последней странице %d записей, ожидалась 1", len(resp.GetNotifications()))
	}
	if resp.GetPage().GetNextPageToken() != "" {
		t.Fatalf("на последней странице выдан токен %q", resp.GetPage().GetNextPageToken())
	}
}

func TestListNotificationsRejectsBadPageToken(t *testing.T) {
	s := newServer(t)

	_, err := s.ListNotifications(context.Background(), &notificationv1.ListNotificationsRequest{
		Page: &commonv1.PageRequest{PageToken: "не-число"},
	})

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("код ошибки = %v, ожидался InvalidArgument", status.Code(err))
	}
}

func TestListNotificationsCapsPageSize(t *testing.T) {
	size, _, err := page(&commonv1.PageRequest{PageSize: 10_000})
	if err != nil {
		t.Fatal(err)
	}

	if size != maxPageSize {
		t.Fatalf("page_size = %d, ожидался предел %d", size, maxPageSize)
	}
}
