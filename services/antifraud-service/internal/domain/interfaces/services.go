package interfaces

import (
	"context"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

type CheckService interface {
	CheckTransfer(ctx context.Context, transfer models.TransferCheck) (*models.Check, error)
}

type RuleService interface {
	ListRules(ctx context.Context) ([]models.RuleConfig, error)
	UpdateRule(ctx context.Context, update models.RuleUpdate) (*models.RuleConfig, error)
}
