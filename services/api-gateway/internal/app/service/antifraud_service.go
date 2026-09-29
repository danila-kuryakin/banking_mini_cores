package service

import (
	"context"
	"time"

	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/antifraud/v1"
)

type AntifraudService struct {
	antifraudClient antifraudv1.AntifraudServiceClient
	timeout         time.Duration
}

func NewAntifraudService(antifraudCli antifraudv1.AntifraudServiceClient, timeout time.Duration) *AntifraudService {
	return &AntifraudService{
		antifraudClient: antifraudCli,
		timeout:         timeout,
	}
}

func (s AntifraudService) ListRules(ctx context.Context) ([]*antifraudv1.Rule, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	listed, err := s.antifraudClient.ListRules(ctx, &antifraudv1.ListRulesRequest{})
	if err != nil {
		return nil, err
	}

	return listed.Rules, nil
}

func (s AntifraudService) UpdateRule(ctx context.Context, in *antifraudv1.UpdateRuleRequest) (*antifraudv1.Rule, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.antifraudClient.UpdateRule(ctx, in)
}
