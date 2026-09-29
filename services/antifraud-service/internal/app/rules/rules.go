package rules

import (
	"context"
	"fmt"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

// Rule - одно правило антифрода; пустая причина означает, что правило не сработало.
type Rule interface {
	Code() string
	Check(ctx context.Context, transfer models.TransferCheck, stats interfaces.CheckStats) (string, error)
}

// definition - как собрать правило из строки af_rules и как проверить его пороги.
type definition struct {
	build    func(params models.RuleParams) Rule
	validate func(params models.RuleParams, currencies models.CurrencySet) error
}

var definitions = map[string]definition{
	domain.RULE_SINGLE_AMOUNT_LIMIT: {
		build:    func(p models.RuleParams) Rule { return singleAmountLimit{limits: p.Limits} },
		validate: validateLimits,
	},
	domain.RULE_VELOCITY: {
		build:    func(p models.RuleParams) Rule { return newVelocity(p) },
		validate: validateVelocity,
	},
	domain.RULE_DAILY_AMOUNT_LIMIT: {
		build:    func(p models.RuleParams) Rule { return dailyAmountLimit{limits: p.Limits} },
		validate: validateLimits,
	},
	domain.RULE_SAME_ACCOUNT: {
		build:    func(models.RuleParams) Rule { return sameAccount{} },
		validate: validateNoParams,
	},
}

// Build собирает включённые правила; неизвестный код в базе - ошибка, а не молчаливый пропуск.
func Build(configs []models.RuleConfig) ([]Rule, error) {
	rules := make([]Rule, 0, len(configs))

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}

		def, ok := definitions[cfg.Code]
		if !ok {
			return nil, fmt.Errorf("rule %q is not implemented", cfg.Code)
		}

		rules = append(rules, def.build(cfg.Params))
	}

	return rules, nil
}

// Validate проверяет пороги до записи в базу, чтобы Build потом не получил негодное правило.
func Validate(code string, params models.RuleParams, currencies models.CurrencySet) error {
	def, ok := definitions[code]
	if !ok {
		return fmt.Errorf("%w: %s", domain.ErrRuleNotFound, code)
	}

	if err := def.validate(params, currencies); err != nil {
		return fmt.Errorf("%w: %s: %v", domain.ErrInvalidRuleParams, code, err)
	}

	return nil
}

// Evaluate прогоняет все правила, а не до первого отказа: в журнал попадает полная картина.
func Evaluate(ctx context.Context, rules []Rule, transfer models.TransferCheck, stats interfaces.CheckStats) ([]models.Hit, error) {
	var hits []models.Hit

	for _, rule := range rules {
		reason, err := rule.Check(ctx, transfer, stats)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", rule.Code(), err)
		}

		if reason != "" {
			hits = append(hits, models.Hit{RuleCode: rule.Code(), Reason: reason})
		}
	}

	return hits, nil
}

func validateLimits(params models.RuleParams, currencies models.CurrencySet) error {
	if params.MaxCount != 0 || params.WindowSeconds != 0 {
		return fmt.Errorf("only limits are allowed")
	}

	for currency, limit := range params.Limits {
		if !currencies.Contains(currency) {
			return fmt.Errorf("currency %q is not in the configured list", currency)
		}

		if limit <= 0 {
			return fmt.Errorf("limit for %s must be positive", currency)
		}
	}

	return nil
}

func validateVelocity(params models.RuleParams, _ models.CurrencySet) error {
	if len(params.Limits) > 0 {
		return fmt.Errorf("limits are not allowed")
	}

	if params.MaxCount <= 0 {
		return fmt.Errorf("max_count must be positive")
	}

	if params.WindowSeconds <= 0 {
		return fmt.Errorf("window_seconds must be positive")
	}

	return nil
}

func validateNoParams(params models.RuleParams, _ models.CurrencySet) error {
	if len(params.Limits) > 0 || params.MaxCount != 0 || params.WindowSeconds != 0 {
		return fmt.Errorf("rule takes no params")
	}

	return nil
}
