package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

type CheckRepo struct {
	db *pgxpool.Pool
}

func NewCheckRepo(db *pgxpool.Pool) *CheckRepo {
	return &CheckRepo{db: db}
}

// SaveCheck перезаписывает результат проверки того же типа: воркер может взять
// заявку повторно после падения, и второй строки записи в таблицу от этого быть не должно.
func (r *CheckRepo) SaveCheck(ctx context.Context, check models.Check) (*models.Check, error) {
	saved, err := scanCheck(r.db.QueryRow(ctx, SAVE_CHECK_QUERY,
		check.ApplicationID,
		check.Type,
		check.Result,
		check.Details,
	))
	if err != nil {
		return nil, fmt.Errorf("save check: %w", err)
	}

	return saved, nil
}
