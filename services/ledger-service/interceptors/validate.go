package interceptors

import (
	"context"
	"strings"

	"google.golang.org/grpc"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/ledger/v1"
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
	case *ledgerv1.TransferRequest:
		if err := notEmpty(in.InitiatorId, domain.ErrInitiatorIDMustBeUUID); err != nil {
			return err
		}

		if err := notEmpty(in.FromAccountId, domain.ErrAccountIDMustBeUUID); err != nil {
			return err
		}

		if err := notEmpty(in.ToAccountId, domain.ErrAccountIDMustBeUUID); err != nil {
			return err
		}

		return validateMoney(in.Amount, in.Currency)
	case *ledgerv1.DepositRequest:
		if err := notEmpty(in.AccountId, domain.ErrAccountIDMustBeUUID); err != nil {
			return err
		}

		if err := notEmpty(in.OfficerId, domain.ErrOfficerIDMustBeUUID); err != nil {
			return err
		}

		return validateMoney(in.Amount, in.Currency)
	case *ledgerv1.GetTransactionRequest:
		return notEmpty(in.TransactionId, domain.ErrTransactionIDMustBeUUID)
	case *ledgerv1.ListTransactionsRequest:
		if err := notEmpty(in.AccountId, domain.ErrAccountIDMustBeUUID); err != nil {
			return err
		}

		return validatePage(in.Limit, in.Offset)
	case *ledgerv1.GetBalanceRequest:
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

func validateMoney(amount int64, currency string) error {
	if amount <= 0 {
		return domain.ErrAmountMustBePositive
	}

	return notEmpty(currency, domain.ErrCurrencyIsRequired)
}
