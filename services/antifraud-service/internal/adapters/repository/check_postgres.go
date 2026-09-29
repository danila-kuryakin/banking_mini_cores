package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

type CheckRepo struct {
	db *pgxpool.Pool
}

func NewCheckRepo(db *pgxpool.Pool) *CheckRepo {
	return &CheckRepo{db: db}
}

// Record держит advisory-блокировку счёта до коммита, иначе параллельные проверки вместе пройдут velocity.
func (r *CheckRepo) Record(
	ctx context.Context,
	transfer models.TransferCheck,
	evaluate interfaces.EvaluateFunc,
) (*models.Check, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("record check: begin: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, LOCK_ACCOUNT_CHECKS_QUERY, transfer.FromAccountID.String()); err != nil {
		return nil, fmt.Errorf("record check: lock account: %w", err)
	}

	hits, err := evaluate(ctx, txStats{tx: tx})
	if err != nil {
		return nil, fmt.Errorf("record check: %w", err)
	}

	checkID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("record check: id: %w", err)
	}

	check := &models.Check{
		ID:       checkID,
		Transfer: transfer,
		Verdict:  models.VerdictFor(hits),
		Hits:     hits,
	}

	triggered := make([]string, 0, len(hits))
	for _, hit := range hits {
		triggered = append(triggered, hit.RuleCode)
	}

	_, err = tx.Exec(ctx, INSERT_CHECK_QUERY,
		check.ID,
		transfer.InitiatorID,
		transfer.FromAccountID,
		transfer.ToAccountID,
		transfer.Amount,
		transfer.Currency,
		check.Verdict,
		triggered,
		check.Reasons(),
		transfer.At,
	)
	if err != nil {
		return nil, fmt.Errorf("record check: insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("record check: commit: %w", err)
	}

	return check, nil
}

// txStats читает историю счёта в транзакции проверки, под её блокировкой.
type txStats struct {
	tx pgx.Tx
}

func (s txStats) CountAllowed(ctx context.Context, accountID uuid.UUID, since time.Time) (int, error) {
	var count int

	if err := s.tx.QueryRow(ctx, COUNT_ALLOWED_QUERY, accountID, since).Scan(&count); err != nil {
		return 0, fmt.Errorf("count allowed checks: %w", err)
	}

	return count, nil
}

func (s txStats) DailyTotal(ctx context.Context, accountID uuid.UUID, currency models.Currency, day time.Time) (int64, error) {
	var total int64

	if err := s.tx.QueryRow(ctx, DAILY_TOTAL_QUERY, accountID, day, currency).Scan(&total); err != nil {
		return 0, fmt.Errorf("daily total: %w", err)
	}

	return total, nil
}
