package interceptors

import (
	"context"
	"strings"

	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/pb/gen/antifraud/v1"
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
	case *antifraudv1.CheckTransferRequest:
		if err := notEmpty(in.InitiatorId, domain.ErrInitiatorIDMustBeUUID); err != nil {
			return err
		}

		if err := notEmpty(in.FromAccountId, domain.ErrAccountIDMustBeUUID); err != nil {
			return err
		}

		if err := notEmpty(in.ToAccountId, domain.ErrAccountIDMustBeUUID); err != nil {
			return err
		}

		if in.Amount <= 0 {
			return domain.ErrAmountMustBePositive
		}

		return notEmpty(in.Currency, domain.ErrCurrencyIsRequired)
	case *antifraudv1.UpdateRuleRequest:
		if err := notEmpty(in.Code, domain.ErrRuleCodeIsRequired); err != nil {
			return err
		}

		if in.Enabled == nil && in.Params == nil {
			return domain.ErrNothingToUpdate
		}

		return notEmpty(in.OfficerId, domain.ErrOfficerIDMustBeUUID)
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
