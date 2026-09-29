package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

type TotalsRepo struct {
	db *pgxpool.Pool
}

func NewTotalsRepo(db *pgxpool.Pool) *TotalsRepo {
	return &TotalsRepo{db: db}
}

// AddTransfer пишет event_id и сумму одной транзакцией: повторная доставка события сумму не удвоит.
func (r *TotalsRepo) AddTransfer(ctx context.Context, transfer models.CompletedTransfer) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("add transfer: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, INSERT_PROCESSED_EVENT_QUERY, transfer.EventID, transfer.EventType)
	if err != nil {
		return false, fmt.Errorf("add transfer: mark event: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return false, nil
	}

	_, err = tx.Exec(ctx, ADD_DAILY_TOTAL_QUERY, transfer.AccountID, transfer.Day, transfer.Currency, transfer.Amount)
	if err != nil {
		return false, fmt.Errorf("add transfer: total: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("add transfer: commit: %w", err)
	}

	return true, nil
}
