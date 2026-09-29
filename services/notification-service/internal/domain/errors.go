package domain

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrPoisonMessage - событие не разобрать, повтор не поможет: оно сразу уходит в DLQ.
	ErrPoisonMessage = errors.New("message cannot be processed")
)

// Ошибки, уходящие наружу gRPC-статусами.
var (
	ErrUserIDMustBeUUID = status.Error(codes.InvalidArgument, "user_id must be a uuid")
	ErrLimitIsNegative  = status.Error(codes.InvalidArgument, "limit must not be negative")
	ErrOffsetIsNegative = status.Error(codes.InvalidArgument, "offset must not be negative")

	ErrListFailed = status.Error(codes.Internal, "failed to list notifications")
)
