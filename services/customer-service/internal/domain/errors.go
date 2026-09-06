package domain

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrCustomerExists              = errors.New("customer already exists")
	ErrCustomerNotFound            = errors.New("customer not found")
	ErrCountCustomersWithoutStatus = errors.New("count customers without status")
)

var (
	ErrUserIDMustBeUUID      = status.Error(codes.InvalidArgument, "user_id must be a uuid")
	ErrCustomerIDMustBeUUID  = status.Error(codes.InvalidArgument, "customer_id must be a uuid")
	ErrCustomerAlreadyExists = status.Error(codes.AlreadyExists, "customer already exists for this user")
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
	ErrProfileLocked = status.Error(codes.FailedPrecondition, "profile cannot be edited after kyc has started")
)
