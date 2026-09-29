package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

type RuleRepository interface {
	ListRules(ctx context.Context) ([]models.RuleConfig, error)
	UpdateRule(ctx context.Context, update models.RuleUpdate) (*models.RuleConfig, error)
}

// CheckStats - история счёта, по которой судят правила; читается внутри транзакции проверки.
type CheckStats interface {
	CountAllowed(ctx context.Context, accountID uuid.UUID, since time.Time) (int, error)
	DailyTotal(ctx context.Context, accountID uuid.UUID, currency models.Currency, day time.Time) (int64, error)
}

// EvaluateFunc прогоняет правила по статистике и возвращает сработавшие.
type EvaluateFunc func(ctx context.Context, stats CheckStats) ([]models.Hit, error)

type CheckRepository interface {
	// Record под блокировкой счёта-источника считает вердикт через evaluate и пишет его в af_checks.
	Record(ctx context.Context, transfer models.TransferCheck, evaluate EvaluateFunc) (*models.Check, error)
}

type TotalsRepository interface {
	// AddTransfer прибавляет перевод к дневной сумме; повтор event_id - no-op и false.
	AddTransfer(ctx context.Context, transfer models.CompletedTransfer) (bool, error)
}
