package service

import (
	"context"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/app/rules"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

type RuleService struct {
	rules      interfaces.RuleRepository
	currencies models.CurrencySet
}

func NewRuleService(rules interfaces.RuleRepository, currencies models.CurrencySet) *RuleService {
	return &RuleService{rules: rules, currencies: currencies}
}

func (s *RuleService) ListRules(ctx context.Context) ([]models.RuleConfig, error) {
	return s.rules.ListRules(ctx)
}

// UpdateRule проверяет новые пороги до записи, чтобы проверка перевода не споткнулась о негодное правило.
func (s *RuleService) UpdateRule(ctx context.Context, update models.RuleUpdate) (*models.RuleConfig, error) {
	if update.Params != nil {
		if err := rules.Validate(update.Code, *update.Params, s.currencies); err != nil {
			return nil, err
		}
	}

	return s.rules.UpdateRule(ctx, update)
}
