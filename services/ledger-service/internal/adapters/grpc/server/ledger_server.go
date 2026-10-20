package server

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/pb/gen/ledger/v1"
)

type LedgerServer struct {
	ledgerv1.UnimplementedLedgerServiceServer
	service *service.Service
	log     *slog.Logger
}

func NewLedgerServer(service *service.Service, log *slog.Logger) *LedgerServer {
	return &LedgerServer{service: service, log: log}
}

func (s *LedgerServer) Transfer(ctx context.Context, in *ledgerv1.TransferRequest) (*ledgerv1.Transaction, error) {
	initiatorID, err := uuid.Parse(in.InitiatorId)
	if err != nil {
		return nil, domain.ErrInitiatorIDMustBeUUID
	}

	from, err := uuid.Parse(in.FromAccountId)
	if err != nil {
		return nil, domain.ErrAccountIDMustBeUUID
	}

	var to uuid.UUID

	if in.ToAccountNumber == "" {
		to, err = uuid.Parse(in.ToAccountId)
		if err != nil {
			return nil, domain.ErrAccountIDMustBeUUID
		}
	} else if len(in.ToAccountNumber) != domain.ACCOUNT_NUMBER_LENGTH || strings.Trim(in.ToAccountNumber, "0123456789") != "" {
		return nil, domain.ErrAccountNumberIsInvalid
	}

	tx, err := s.service.Ledger.Transfer(ctx, models.TransferCommand{
		InitiatorID: initiatorID,
		From:        from,
		To:          to,
		Amount:      in.Amount,
		Currency:    in.Currency,
		ToNumber:    in.ToAccountNumber,
	})
	if err != nil {
		return nil, s.fail("failed to transfer", err, tx, domain.ErrTransferFailed, "from_account_id", in.FromAccountId)
	}

	return transactionToProto(tx), nil
}

func (s *LedgerServer) Deposit(ctx context.Context, in *ledgerv1.DepositRequest) (*ledgerv1.Transaction, error) {
	accountID, err := uuid.Parse(in.AccountId)
	if err != nil {
		return nil, domain.ErrAccountIDMustBeUUID
	}

	officerID, err := uuid.Parse(in.OfficerId)
	if err != nil {
		return nil, domain.ErrOfficerIDMustBeUUID
	}

	tx, err := s.service.Ledger.Deposit(ctx, models.DepositCommand{
		AccountID: accountID,
		Amount:    in.Amount,
		Currency:  in.Currency,
		OfficerID: officerID,
	})
	if err != nil {
		return nil, s.fail("failed to deposit", err, tx, domain.ErrDepositFailed, "account_id", in.AccountId)
	}

	return transactionToProto(tx), nil
}

func (s *LedgerServer) GetTransaction(ctx context.Context, in *ledgerv1.GetTransactionRequest) (*ledgerv1.Transaction, error) {
	transactionId, err := uuid.Parse(in.TransactionId)
	if err != nil {
		return nil, domain.ErrTransactionIDMustBeUUID
	}

	tx, err := s.service.Ledger.GetTransaction(ctx, transactionId)
	if err != nil {
		return nil, s.fail("failed to get transaction", err, nil, domain.ErrGetFailed, "transaction_id", in.TransactionId)
	}

	return transactionToProto(tx), nil
}

func (s *LedgerServer) ListTransactions(ctx context.Context, in *ledgerv1.ListTransactionsRequest) (*ledgerv1.ListTransactionsResponse, error) {
	var userID *uuid.UUID

	if in.UserId != "" {
		parsed, err := uuid.Parse(in.UserId)
		if err != nil {
			return nil, domain.ErrUserIDMustBeUUID
		}

		userID = &parsed
	}

	accountID, err := uuid.Parse(in.AccountId)
	if err != nil {
		return nil, domain.ErrAccountIDMustBeUUID
	}

	transactions, err := s.service.Ledger.ListTransactions(ctx, userID, accountID, models.Page{
		Limit:  int(in.Limit),
		Offset: int(in.Offset),
	})
	if err != nil {
		return nil, s.fail("failed to list transactions", err, nil, domain.ErrListFailed, "account_id", in.AccountId)
	}

	return &ledgerv1.ListTransactionsResponse{Transactions: transactionsToProto(transactions)}, nil
}

func (s *LedgerServer) GetBalance(ctx context.Context, in *ledgerv1.GetBalanceRequest) (*ledgerv1.GetBalanceResponse, error) {
	accountID, err := uuid.Parse(in.AccountId)
	if err != nil {
		return nil, domain.ErrAccountIDMustBeUUID
	}

	balance, err := s.service.Ledger.GetBalance(ctx, accountID)
	if err != nil {
		return nil, s.fail("failed to get balance", err, nil, domain.ErrBalanceFailed, "account_id", in.AccountId)
	}

	return &ledgerv1.GetBalanceResponse{
		AccountId: balance.AccountID.String(),
		Amount:    balance.Amount,
		Currency:  string(balance.Currency),
	}, nil
}

// fail переводит доменную ошибку в gRPC-статус; у отклонённой транзакции её id уходит в текст ошибки.
func (s *LedgerServer) fail(msg string, err error, tx *models.Transaction, fallback error, logArgs ...any) error {
	switch {
	case errors.Is(err, domain.ErrInsufficientFunds), errors.Is(err, domain.ErrTransferDenied):
		if tx != nil {
			return status.Errorf(codes.FailedPrecondition, "transaction %s declined: %s", tx.ID, tx.DeclineReason)
		}

		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrTransactionNotFound):
		return domain.ErrTransactionMissing
	case errors.Is(err, domain.ErrBalanceNotFound):
		return domain.ErrBalanceMissing
	case errors.Is(err, domain.ErrAccountNotFound):
		return domain.ErrAccountMissing
	case errors.Is(err, domain.ErrAccountNotOwned):
		return domain.ErrForeignAccount
	case errors.Is(err, domain.ErrAccountNotActive):
		return domain.ErrAccountIsNotActive
	case errors.Is(err, domain.ErrCurrencyMismatch):
		return domain.ErrCurrencyIsMismatch
	case errors.Is(err, domain.ErrCurrencyNotAllowed):
		return domain.ErrCurrencyIsNotValid
	case errors.Is(err, domain.ErrSameAccount):
		return domain.ErrSameAccountTransfer
	case errors.Is(err, domain.ErrFraudCheckFailed):
		s.log.Error(msg, append(logArgs, "error", err)...)

		return domain.ErrAntifraudIsDown
	}

	s.log.Error(msg, append(logArgs, "error", err)...)

	return fallback
}
