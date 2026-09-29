package service

import (
	"context"
	"time"

	notificationv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/notification/v1"
)

type NotificationService struct {
	notificationClient notificationv1.NotificationServiceClient
	timeout            time.Duration
}

func NewNotificationService(notificationCli notificationv1.NotificationServiceClient, timeout time.Duration) *NotificationService {
	return &NotificationService{
		notificationClient: notificationCli,
		timeout:            timeout,
	}
}

func (s NotificationService) ListNotifications(
	ctx context.Context,
	in *notificationv1.ListNotificationsRequest,
) ([]*notificationv1.Notification, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	listed, err := s.notificationClient.ListNotifications(ctx, in)
	if err != nil {
		return nil, err
	}

	return listed.Notifications, nil
}
