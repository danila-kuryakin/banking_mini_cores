package domain

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrCustomerExists   = errors.New("customer already exists")
	ErrCustomerNotFound = errors.New("customer not found")
	// ErrStatusConflict - статус клиента изменился между проверкой перехода и
	// записью. Ловится условием на исходный статус в UPDATE.
	ErrStatusConflict = errors.New("customer status changed concurrently")
)

var (
	ErrUserIDMustBeUUID      = status.Error(codes.InvalidArgument, "user_id must be a uuid")
	ErrCustomerAlreadyExists = status.Error(codes.AlreadyExists, "customer already exists for this user")
	ErrProfileRequired       = status.Error(codes.InvalidArgument, "profile is required")
	ErrCustomerMissing       = status.Error(codes.NotFound, "customer not found")
	ErrCreateProfileFailed   = status.Error(codes.Internal, "failed to create profile")
	ErrListCustomersFailed   = status.Error(codes.Internal, "failed to list customers")
)

var (
	ErrFirstNameIsTooLong   = status.Error(codes.InvalidArgument, "first_name is longer than 100 characters")
	ErrLastNameIsTooLong    = status.Error(codes.InvalidArgument, "last_name is longer than 100 characters")
	ErrCitizenshipIsTooLong = status.Error(codes.InvalidArgument, "citizenship is longer than 64 characters")
	ErrPhoneIsNotValid      = status.Error(codes.InvalidArgument, "phone must contain 7 to 15 digits with an optional leading plus")
	ErrBirthDateIsNotValid  = status.Error(codes.InvalidArgument, "birth_date is not a calendar date")
	ErrBirthDateInFuture    = status.Error(codes.InvalidArgument, "birth_date is in the future")
	ErrBirthDateIsTooOld    = status.Error(codes.InvalidArgument, "birth_date is more than 120 years ago")
	ErrLimitIsNegative      = status.Error(codes.InvalidArgument, "limit must not be negative")
	ErrOffsetIsNegative     = status.Error(codes.InvalidArgument, "offset must not be negative")
)

var (
	ErrProfileLocked = status.Error(codes.FailedPrecondition, "profile can only be edited while the customer is in the new status")
)

var (
	ErrStatusRequired            = status.Error(codes.InvalidArgument, "status is required and must be a known customer status")
	ErrActorIDMustBeUUID         = status.Error(codes.InvalidArgument, "actor_id must be a uuid")
	ErrReasonIsTooLong           = status.Error(codes.InvalidArgument, "reason is longer than 500 characters")
	ErrReasonRequired            = status.Error(codes.InvalidArgument, "reason is required to reject a customer")
	ErrStatusTransitionForbidden = status.Error(codes.FailedPrecondition, "customer cannot move to the requested status from the current one")
	ErrStatusChanged             = status.Error(codes.Aborted, "customer status changed while the request was in flight")
	// Переход в blocked разрешён из любого статуса, но выставлять его вправе
	// только админ, а проверки роли в сервисе пока нет: до неё лучше не уметь
	// блокировать вовсе, чем уметь без проверки.
	ErrBlockingNotImplemented = status.Error(codes.Unimplemented, "blocking a customer is not implemented yet")
	ErrSetStatusFailed        = status.Error(codes.Internal, "failed to set customer status")
)
