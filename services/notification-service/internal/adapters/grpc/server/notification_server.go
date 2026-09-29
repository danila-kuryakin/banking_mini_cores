package server

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
	notificationv1 "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/pb/gen/notification/v1"
)

type NotificationServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	service *service.Service
	log     *slog.Logger
}

func NewNotificationServer(service *service.Service, log *slog.Logger) *NotificationServer {
	return &NotificationServer{service: service, log: log}
}

func (s *NotificationServer) ListNotifications(
	ctx context.Context,
	in *notificationv1.ListNotificationsRequest,
) (*notificationv1.ListNotificationsResponse, error) {
	userID, err := uuid.Parse(in.UserId)
	if err != nil {
		return nil, domain.ErrUserIDMustBeUUID
	}

	notifications, err := s.service.Notifications.ListNotifications(ctx, userID, models.Page{
		Limit:  int(in.Limit),
		Offset: int(in.Offset),
	})
	if err != nil {
		s.log.Error("failed to list notifications", "user_id", in.UserId, "error", err)

		return nil, domain.ErrListFailed
	}

	out := make([]*notificationv1.Notification, 0, len(notifications))
	for _, n := range notifications {
		out = append(out, &notificationv1.Notification{
			Id:        n.ID.String(),
			UserId:    n.UserID.String(),
			EventId:   n.EventID.String(),
			EventType: n.EventType,
			Text:      n.Text,
			CreatedAt: timestamppb.New(n.CreatedAt),
		})
	}

	return &notificationv1.ListNotificationsResponse{Notifications: out}, nil
}
