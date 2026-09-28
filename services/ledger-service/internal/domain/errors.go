package domain

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrBalanceNotFound     = errors.New("account has no ledger movements")
	// ErrInsufficientFunds - перевод отклонён и записан со статусом declined.
	ErrInsufficientFunds = errors.New("insufficient funds")
	// ErrTransferDenied - антифрод запретил перевод, он записан со статусом declined.
	ErrTransferDenied = errors.New("transfer denied by antifraud")
	// ErrCurrencyNotAllowed - валюты нет в списке currencies конфига.
	ErrCurrencyNotAllowed = errors.New("currency is not allowed")
	// ErrCurrencyMismatch - валюта операции не совпадает с валютой счёта.
	ErrCurrencyMismatch = errors.New("currency does not match the account currency")
	ErrSameAccount      = errors.New("source and destination accounts are the same")
)

// Ошибки, приходящие от account-service.
var (
	ErrAccountNotFound  = errors.New("account not found")
	ErrAccountNotActive = errors.New("account is not active")
	// ErrAccountNotOwned - инициатор перевода не владеет счётом-источником.
	ErrAccountNotOwned = errors.New("account belongs to another user")
)

// Ошибки, уходящие наружу gRPC-статусами.
var (
	ErrAccountIDMustBeUUID     = status.Error(codes.InvalidArgument, "account_id must be a uuid")
	ErrTransactionIDMustBeUUID = status.Error(codes.InvalidArgument, "transaction_id must be a uuid")
	ErrUserIDMustBeUUID        = status.Error(codes.InvalidArgument, "user_id must be a uuid")
	ErrInitiatorIDMustBeUUID   = status.Error(codes.InvalidArgument, "initiator_id must be a uuid")
	ErrOfficerIDMustBeUUID     = status.Error(codes.InvalidArgument, "officer_id must be a uuid")
	ErrAmountMustBePositive    = status.Error(codes.InvalidArgument, "amount must be positive")
	ErrCurrencyIsRequired      = status.Error(codes.InvalidArgument, "currency is required")
	ErrLimitIsNegative         = status.Error(codes.InvalidArgument, "limit must not be negative")
	ErrOffsetIsNegative        = status.Error(codes.InvalidArgument, "offset must not be negative")
	ErrSameAccountTransfer     = status.Error(codes.InvalidArgument, "cannot transfer to the same account")

	ErrAccountMissing     = status.Error(codes.NotFound, "account not found")
	ErrTransactionMissing = status.Error(codes.NotFound, "transaction not found")
	ErrBalanceMissing     = status.Error(codes.NotFound, "account has no ledger movements")
	ErrForeignAccount     = status.Error(codes.PermissionDenied, "source account belongs to another user")
	ErrAccountIsNotActive = status.Error(codes.FailedPrecondition, "both accounts must be active")
	ErrCurrencyIsMismatch = status.Error(codes.FailedPrecondition, "currency does not match the account currency")

	ErrTransferFailed = status.Error(codes.Internal, "failed to transfer")
	ErrDepositFailed  = status.Error(codes.Internal, "failed to deposit")
	ErrGetFailed      = status.Error(codes.Internal, "failed to get transaction")
	ErrListFailed     = status.Error(codes.Internal, "failed to list transactions")
	ErrBalanceFailed  = status.Error(codes.Internal, "failed to get balance")
)
