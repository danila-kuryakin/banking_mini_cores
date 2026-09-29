package interceptors

import (
	"context"
	"strings"
	"unicode/utf8"

	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/domain"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/account-service/internal/pb/gen/account/v1"
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
	case *accountv1.OpenAccountRequest:
		return notEmpty(in.UserId, domain.ErrUserIDMustBeUUID)
	case *accountv1.GetAccountRequest:
		return notEmpty(in.AccountId, domain.ErrAccountIDMustBeUUID)
	case *accountv1.ListAccountsRequest:
		if err := notEmpty(in.UserId, domain.ErrUserIDMustBeUUID); err != nil {
			return err
		}

		return validatePage(in.Limit, in.Offset)
	case *accountv1.ListStatusHistoryRequest:
		if err := notEmpty(in.AccountId, domain.ErrAccountIDMustBeUUID); err != nil {
			return err
		}

		return validatePage(in.Limit, in.Offset)
	case *accountv1.BlockAccountRequest:
		if err := validateOfficerAction(in.AccountId, in.OfficerId); err != nil {
			return err
		}

		if strings.TrimSpace(in.Reason) == "" {
			return domain.ErrReasonIsRequired
		}

		return validateReason(in.Reason)
	case *accountv1.UnblockAccountRequest:
		if err := validateOfficerAction(in.AccountId, in.OfficerId); err != nil {
			return err
		}

		return validateReason(in.Comment)
	case *accountv1.CloseAccountRequest:
		if err := notEmpty(in.UserId, domain.ErrUserIDMustBeUUID); err != nil {
			return err
		}

		return notEmpty(in.AccountId, domain.ErrAccountIDMustBeUUID)
	case *accountv1.GetBalanceRequest:
		if err := notEmpty(in.UserId, domain.ErrUserIDMustBeUUID); err != nil {
			return err
		}

		return notEmpty(in.AccountId, domain.ErrAccountIDMustBeUUID)
	default:
		return nil
	}
}

func notEmpty(value string, err error) error {
	if strings.TrimSpace(value) == "" {
		return err
	}

	return nil
}

func validateOfficerAction(accountID, officerID string) error {
	if err := notEmpty(accountID, domain.ErrAccountIDMustBeUUID); err != nil {
		return err
	}

	return notEmpty(officerID, domain.ErrOfficerIDMustBeUUID)
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

func validateReason(reason string) error {
	if utf8.RuneCountInString(reason) > domain.REASON_MAX_LENGTH {
		return domain.ErrReasonIsTooLong
	}

	return nil
}
