package interceptors

import (
	"context"
	"unicode/utf8"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/pb/gen/customer/v1"
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
	case *customerv1.CreateProfileRequest:
		return validateUserID(in.UserId)
	case *customerv1.GetCustomerRequest:
		return validateUserID(in.UserId)
	case *customerv1.UpdateProfileRequest:
		return validateUserID(in.UserId)
	case *customerv1.GetCustomerStatusRequest:
		return validateUserID(in.UserId)
	case *customerv1.SetStatusRequest:
		return validateSetStatus(in)
	case *customerv1.ListCustomersRequest:
		return validatePage(in.Limit, in.Offset)
	default:
		return nil
	}
}

func validateUserID(userID string) error {
	if _, err := uuid.Parse(userID); err != nil {
		return domain.ErrUserIDMustBeUUID
	}

	return nil
}

func validateSetStatus(in *customerv1.SetStatusRequest) error {
	if err := validateUserID(in.UserId); err != nil {
		return err
	}

	if in.Status == customerv1.CustomerStatus_UNSPECIFIED {
		return domain.ErrStatusRequired
	}

	if in.ActorId != "" {
		if _, err := uuid.Parse(in.ActorId); err != nil {
			return domain.ErrActorIDMustBeUUID
		}
	}

	if utf8.RuneCountInString(in.Reason) > domain.REASON_MAX_LENGTH {
		return domain.ErrReasonIsTooLong
	}

	return nil
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
