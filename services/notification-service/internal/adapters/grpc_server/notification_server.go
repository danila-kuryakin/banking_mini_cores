package grpc_server

import (
	"context"
	"log/slog"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/adapters/memory"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/notification"
	commonv1 "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/pb/gen/common/v1"
	notificationv1 "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/pb/gen/notification/v1"
)

// Размеры страницы. ListNotifications - отладочный метод поверх хранилища в
// памяти, поэтому предел скромный: страница на тысячу записей тут никому
// не нужна, а память сервису не бесконечная.
const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// NotificationServer отдаёт то, что консьюмер сложил в хранилище.
//
// Метод один и он для отладки: настоящая доставка уведомлений живёт в
// консьюмере, а gRPC тут - способ посмотреть, что событие доехало, не читая
// топик руками через kafka-console-consumer.
type NotificationServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	repo *memory.Repository
	log  *slog.Logger
}

func NewNotification(repo *memory.Repository, log *slog.Logger) *NotificationServer {
	return &NotificationServer{
		repo: repo,
		log:  log,
	}
}

func (s *NotificationServer) ListNotifications(ctx context.Context, in *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	size, offset, err := page(in.GetPage())
	if err != nil {
		return nil, err
	}

	filter := notification.Filter{
		CustomerID: in.GetCustomerId(),
		Status:     statusFromPB(in.GetStatus()),
	}

	items, total, err := s.repo.Notification.List(ctx, filter, offset, size)
	if err != nil {
		s.log.Error("list notifications", slog.Any("error", err))
		return nil, status.Error(codes.Internal, "failed to list notifications")
	}

	out := make([]*notificationv1.Notification, 0, len(items))
	for _, n := range items {
		out = append(out, toPB(n))
	}

	// Токен следующей страницы - это просто смещение. Для отладочного метода
	// поверх памяти курсор не нужен, а число читается глазами.
	var next string
	if offset+len(items) < total {
		next = strconv.Itoa(offset + len(items))
	}

	return &notificationv1.ListNotificationsResponse{
		Notifications: out,
		Page: &commonv1.PageResponse{
			NextPageToken: next,
			TotalSize:     int32(total),
		},
	}, nil
}

// page разбирает постраничность запроса в размер и смещение.
func page(p *commonv1.PageRequest) (size, offset int, err error) {
	size = defaultPageSize

	if p == nil {
		return size, 0, nil
	}

	if p.GetPageSize() > 0 {
		size = int(p.GetPageSize())
	}

	if size > maxPageSize {
		size = maxPageSize
	}

	token := p.GetPageToken()
	if token == "" {
		return size, 0, nil
	}

	offset, convErr := strconv.Atoi(token)
	if convErr != nil || offset < 0 {
		// InvalidArgument, а не молчаливый сброс на нулевую страницу: клиент,
		// приславший чужой токен, должен об этом узнать.
		return 0, 0, status.Errorf(codes.InvalidArgument, "invalid page_token %q", token)
	}

	return size, offset, nil
}

func toPB(n notification.Notification) *notificationv1.Notification {
	out := &notificationv1.Notification{
		NotificationId: n.ID,
		CustomerId:     n.CustomerID,
		EventId:        n.EventID,
		EventType:      n.EventType,
		Channel:        channelToPB(n.Channel),
		Status:         statusToPB(n.Status),
		Subject:        n.Subject,
		Body:           n.Body,
		Attempts:       n.Attempts,
		CreatedAt:      timestamppb.New(n.CreatedAt),
	}

	// Неотправленное уведомление не должно приезжать с sent_at = 1970 год:
	// у нулевого времени в protobuf нет отдельного представления, поэтому
	// поле просто остаётся пустым.
	if !n.SentAt.IsZero() {
		out.SentAt = timestamppb.New(n.SentAt)
	}

	return out
}

func channelToPB(c notification.Channel) notificationv1.NotificationChannel {
	switch c {
	case notification.ChannelEmail:
		return notificationv1.NotificationChannel_NOTIFICATION_CHANNEL_EMAIL
	case notification.ChannelSMS:
		return notificationv1.NotificationChannel_NOTIFICATION_CHANNEL_SMS
	case notification.ChannelPush:
		return notificationv1.NotificationChannel_NOTIFICATION_CHANNEL_PUSH
	default:
		return notificationv1.NotificationChannel_NOTIFICATION_CHANNEL_UNSPECIFIED
	}
}

func statusToPB(s notification.Status) notificationv1.NotificationStatus {
	switch s {
	case notification.StatusPending:
		return notificationv1.NotificationStatus_NOTIFICATION_STATUS_PENDING
	case notification.StatusSent:
		return notificationv1.NotificationStatus_NOTIFICATION_STATUS_SENT
	case notification.StatusFailed:
		return notificationv1.NotificationStatus_NOTIFICATION_STATUS_FAILED
	default:
		return notificationv1.NotificationStatus_NOTIFICATION_STATUS_UNSPECIFIED
	}
}

// statusFromPB переводит фильтр запроса в доменный статус. UNSPECIFIED
// означает "не фильтровать", поэтому превращается в пустую строку.
func statusFromPB(s notificationv1.NotificationStatus) notification.Status {
	switch s {
	case notificationv1.NotificationStatus_NOTIFICATION_STATUS_PENDING:
		return notification.StatusPending
	case notificationv1.NotificationStatus_NOTIFICATION_STATUS_SENT:
		return notification.StatusSent
	case notificationv1.NotificationStatus_NOTIFICATION_STATUS_FAILED:
		return notification.StatusFailed
	default:
		return ""
	}
}
