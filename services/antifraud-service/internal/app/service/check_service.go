package service

import (
	"context"
	"fmt"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/app/rules"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

type CheckService struct {
	checks interfaces.CheckRepository
	rules  interfaces.RuleRepository
}

func NewCheckService(checks interfaces.CheckRepository, rules interfaces.RuleRepository) *CheckService {
	return &CheckService{checks: checks, rules: rules}
}

// CheckTransfer читает правила при каждой проверке: правка офицера действует со следующего перевода.
func (s *CheckService) CheckTransfer(ctx context.Context, transfer models.TransferCheck) (*models.Check, error) {
	configs, err := s.rules.ListRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("check transfer: %w", err)
	}

	active, err := rules.Build(configs)
	if err != nil {
		return nil, fmt.Errorf("check transfer: %w", err)
	}

	transfer.At = time.Now().UTC()

	return s.checks.Record(ctx, transfer, func(ctx context.Context, stats interfaces.CheckStats) ([]models.Hit, error) {
		return rules.Evaluate(ctx, active, transfer, stats)
	})
}
