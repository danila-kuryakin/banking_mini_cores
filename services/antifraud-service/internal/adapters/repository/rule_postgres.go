package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

type RuleRepo struct {
	db *pgxpool.Pool
}

func NewRuleRepo(db *pgxpool.Pool) *RuleRepo {
	return &RuleRepo{db: db}
}

func (r *RuleRepo) ListRules(ctx context.Context) ([]models.RuleConfig, error) {
	rows, err := r.db.Query(ctx, LIST_RULES_QUERY)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	rules := make([]models.RuleConfig, 0, 4)

	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, fmt.Errorf("scan rules: %w", err)
		}

		rules = append(rules, *rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rules: %w", err)
	}

	return rules, nil
}

func (r *RuleRepo) UpdateRule(ctx context.Context, update models.RuleUpdate) (*models.RuleConfig, error) {
	var params []byte

	if update.Params != nil {
		encoded, err := json.Marshal(update.Params)
		if err != nil {
			return nil, fmt.Errorf("update rule: params: %w", err)
		}

		params = encoded
	}

	rule, err := scanRule(r.db.QueryRow(ctx, UPDATE_RULE_QUERY, update.Code, update.Enabled, params, update.OfficerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRuleNotFound
		}

		return nil, fmt.Errorf("update rule: %w", err)
	}

	return rule, nil
}

type row interface {
	Scan(dest ...any) error
}

func scanRule(r row) (*models.RuleConfig, error) {
	var (
		rule      models.RuleConfig
		params    []byte
		updatedAt time.Time
	)

	if err := r.Scan(&rule.Code, &rule.Description, &rule.Enabled, &params, &rule.UpdatedBy, &updatedAt); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(params, &rule.Params); err != nil {
		return nil, fmt.Errorf("rule %s params: %w", rule.Code, err)
	}

	rule.UpdatedAt = updatedAt.UTC()

	return &rule, nil
}
