package domain

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrRuleNotFound = errors.New("rule not found")
	// ErrInvalidRuleParams - пороги не подходят правилу: отрицательные, не та валюта, лишние поля.
	ErrInvalidRuleParams = errors.New("invalid rule params")
	// ErrPoisonMessage - событие не разобрать, повтор не поможет.
	ErrPoisonMessage = errors.New("message cannot be processed")
)

// Ошибки, уходящие наружу gRPC-статусами.
var (
	ErrInitiatorIDMustBeUUID = status.Error(codes.InvalidArgument, "initiator_id must be a uuid")
	ErrAccountIDMustBeUUID   = status.Error(codes.InvalidArgument, "account_id must be a uuid")
	ErrOfficerIDMustBeUUID   = status.Error(codes.InvalidArgument, "officer_id must be a uuid")
	ErrAmountMustBePositive  = status.Error(codes.InvalidArgument, "amount must be positive")
	ErrCurrencyIsRequired    = status.Error(codes.InvalidArgument, "currency is required")
	ErrRuleCodeIsRequired    = status.Error(codes.InvalidArgument, "rule code is required")
	ErrNothingToUpdate       = status.Error(codes.InvalidArgument, "either enabled or params must be set")

	ErrRuleMissing = status.Error(codes.NotFound, "rule not found")

	ErrCheckFailed      = status.Error(codes.Internal, "failed to check transfer")
	ErrListRulesFailed  = status.Error(codes.Internal, "failed to list rules")
	ErrUpdateRuleFailed = status.Error(codes.Internal, "failed to update rule")
)
