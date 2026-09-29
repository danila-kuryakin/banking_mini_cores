package interceptors

import (
	"context"
	"strings"

	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain"
	notificationv1 "github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/pb/gen/notification/v1"
)

func Validate() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := validate(req); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func validate(req any) error {
	switch in := req.(type) {
	case *notificationv1.ListNotificationsRequest:
		if strings.TrimSpace(in.UserId) == "" {
			return domain.ErrUserIDMustBeUUID
		}

		return validatePage(in.Limit, in.Offset)
	default:
		return nil
	}
}

func validatePage(limit, offset int32) error {
	switch {
	case limit < 0:
		return domain.ErrLimitIsNegative
	case offset < 0:
		return domain.ErrOffsetIsNegative
	default:
		return nil
	}
}
