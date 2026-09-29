package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/pb/gen/antifraud/v1"
)

type AntifraudServer struct {
	antifraudv1.UnimplementedAntifraudServiceServer
	service *service.Service
	log     *slog.Logger
}

func NewAntifraudServer(service *service.Service, log *slog.Logger) *AntifraudServer {
	return &AntifraudServer{service: service, log: log}
}

func (s *AntifraudServer) CheckTransfer(ctx context.Context, in *antifraudv1.CheckTransferRequest) (*antifraudv1.CheckTransferResponse, error) {
	initiatorID, err := uuid.Parse(in.InitiatorId)
	if err != nil {
		return nil, domain.ErrInitiatorIDMustBeUUID
	}

	fromAccountID, err := uuid.Parse(in.FromAccountId)
	if err != nil {
		return nil, domain.ErrAccountIDMustBeUUID
	}

	toAccountID, err := uuid.Parse(in.ToAccountId)
	if err != nil {
		return nil, domain.ErrAccountIDMustBeUUID
	}

	currency, err := models.ParseCurrency(in.Currency)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	check, err := s.service.Checks.CheckTransfer(ctx, models.TransferCheck{
		InitiatorID:   initiatorID,
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        in.Amount,
		Currency:      currency,
	})
	if err != nil {
		return nil, s.fail("failed to check transfer", err, domain.ErrCheckFailed, "from_account_id", in.FromAccountId)
	}

	if check.Verdict == models.VERDICT_DENY {
		s.log.Info("transfer denied", "check_id", check.ID, "from_account_id", in.FromAccountId, "reasons", check.Reasons())
	}

	return &antifraudv1.CheckTransferResponse{
		CheckId: check.ID.String(),
		Verdict: verdictToProto[check.Verdict],
		Reasons: check.Reasons(),
	}, nil
}

func (s *AntifraudServer) ListRules(ctx context.Context, _ *antifraudv1.ListRulesRequest) (*antifraudv1.ListRulesResponse, error) {
	rules, err := s.service.Rules.ListRules(ctx)
	if err != nil {
		return nil, s.fail("failed to list rules", err, domain.ErrListRulesFailed)
	}

	out := make([]*antifraudv1.Rule, 0, len(rules))
	for i := range rules {
		out = append(out, ruleToProto(&rules[i]))
	}

	return &antifraudv1.ListRulesResponse{Rules: out}, nil
}

func (s *AntifraudServer) UpdateRule(ctx context.Context, in *antifraudv1.UpdateRuleRequest) (*antifraudv1.Rule, error) {
	officerID, err := uuid.Parse(in.OfficerId)
	if err != nil {
		return nil, domain.ErrOfficerIDMustBeUUID
	}

	params, err := paramsFromProto(in.Params)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	rule, err := s.service.Rules.UpdateRule(ctx, models.RuleUpdate{
		Code:      in.Code,
		Enabled:   in.Enabled,
		Params:    params,
		OfficerID: officerID,
	})
	if err != nil {
		return nil, s.fail("failed to update rule", err, domain.ErrUpdateRuleFailed, "code", in.Code)
	}

	s.log.Info("rule updated", "code", rule.Code, "enabled", rule.Enabled, "officer_id", in.OfficerId)

	return ruleToProto(rule), nil
}

// fail переводит доменную ошибку в gRPC-статус, неизвестную - логирует и прячет за fallback.
func (s *AntifraudServer) fail(msg string, err error, fallback error, logArgs ...any) error {
	switch {
	case errors.Is(err, domain.ErrRuleNotFound):
		return domain.ErrRuleMissing
	case errors.Is(err, domain.ErrInvalidRuleParams):
		return status.Error(codes.InvalidArgument, err.Error())
	}

	s.log.Error(msg, append(logArgs, "error", err)...)

	return fallback
}
