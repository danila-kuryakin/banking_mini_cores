package domain

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrAccountNotFound     = errors.New("account not found")
	ErrAccountLimitReached = errors.New("account limit per currency reached")
	ErrStatusConflict      = errors.New("account status does not allow the transition")
	ErrCurrencyNotAllowed  = errors.New("currency is not allowed")
	ErrBalanceNotZero      = errors.New("account balance is not zero")
)

var (
	ErrCustomerNotFound  = errors.New("customer not found")
	ErrCustomerNotActive = errors.New("customer is not active")
)

var (
	ErrAccountIDMustBeUUID = status.Error(codes.InvalidArgument, "account_id must be a uuid")
	ErrUserIDMustBeUUID    = status.Error(codes.InvalidArgument, "user_id must be a uuid")
	ErrOfficerIDMustBeUUID = status.Error(codes.InvalidArgument, "officer_id must be a uuid")
	ErrLimitIsNegative     = status.Error(codes.InvalidArgument, "limit must not be negative")
	ErrOffsetIsNegative    = status.Error(codes.InvalidArgument, "offset must not be negative")
	ErrReasonIsRequired    = status.Error(codes.InvalidArgument, "reason is required to block an account")
	ErrReasonIsTooLong     = status.Error(codes.InvalidArgument, "reason is longer than 500 characters")
	ErrCurrencyIsNotValid  = status.Error(codes.InvalidArgument, "currency is not allowed")
	ErrNumberIsInvalid     = status.Error(codes.InvalidArgument, "number must be 20 digits")

	ErrAccountMissing      = status.Error(codes.NotFound, "account not found")
	ErrCustomerMissing     = status.Error(codes.NotFound, "customer not found")
	ErrCustomerIsNotActive = status.Error(codes.FailedPrecondition, "customer must be active to open an account")
	ErrAccountLimit        = status.Error(codes.FailedPrecondition, "customer already has the maximum number of accounts in this currency")
	ErrTransitionForbidden = status.Error(codes.FailedPrecondition, "account cannot move to the requested status")
	ErrBalanceIsNotZero    = status.Error(codes.FailedPrecondition, "only an account with zero balance can be closed")

	ErrOpenFailed    = status.Error(codes.Internal, "failed to open account")
	ErrGetFailed     = status.Error(codes.Internal, "failed to get account")
	ErrListFailed    = status.Error(codes.Internal, "failed to list accounts")
	ErrUpdateFailed  = status.Error(codes.Internal, "failed to update account")
	ErrBalanceFailed = status.Error(codes.Internal, "failed to get account balance")
	ErrHistoryFailed = status.Error(codes.Internal, "failed to list account status history")
)
