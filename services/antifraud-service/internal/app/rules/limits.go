package rules

import (
	"context"
	"fmt"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

// singleAmountLimit запрещает один перевод больше порога валюты.
type singleAmountLimit struct {
	limits map[models.Currency]int64
}

func (singleAmountLimit) Code() string { return domain.RULE_SINGLE_AMOUNT_LIMIT }

func (r singleAmountLimit) Check(_ context.Context, transfer models.TransferCheck, _ interfaces.CheckStats) (string, error) {
	limit, ok := r.limits[transfer.Currency]
	if !ok || transfer.Amount <= limit {
		return "", nil
	}

	return fmt.Sprintf("amount %d exceeds limit %d %s", transfer.Amount, limit, transfer.Currency), nil
}

// dailyAmountLimit запрещает перевод, если с ним сумма проведённых за день UTC превысит порог.
type dailyAmountLimit struct {
	limits map[models.Currency]int64
}

func (dailyAmountLimit) Code() string { return domain.RULE_DAILY_AMOUNT_LIMIT }

func (r dailyAmountLimit) Check(ctx context.Context, transfer models.TransferCheck, stats interfaces.CheckStats) (string, error) {
	limit, ok := r.limits[transfer.Currency]
	if !ok {
		return "", nil
	}

	total, err := stats.DailyTotal(ctx, transfer.FromAccountID, transfer.Currency, models.Day(transfer.At))
	if err != nil {
		return "", err
	}

	if total+transfer.Amount <= limit {
		return "", nil
	}

	return fmt.Sprintf("daily total %d plus amount %d exceeds limit %d %s", total, transfer.Amount, limit, transfer.Currency), nil
}

// velocity запрещает больше maxCount разрешённых переводов со счёта за окно.
type velocity struct {
	maxCount int
	window   time.Duration
}

func newVelocity(params models.RuleParams) velocity {
	return velocity{
		maxCount: params.MaxCount,
		window:   time.Duration(params.WindowSeconds) * time.Second,
	}
}

func (velocity) Code() string { return domain.RULE_VELOCITY }

func (r velocity) Check(ctx context.Context, transfer models.TransferCheck, stats interfaces.CheckStats) (string, error) {
	count, err := stats.CountAllowed(ctx, transfer.FromAccountID, transfer.At.Add(-r.window))
	if err != nil {
		return "", err
	}

	if count < r.maxCount {
		return "", nil
	}

	return fmt.Sprintf("%d transfers in the last %s, limit %d", count, r.window, r.maxCount), nil
}

// sameAccount запрещает перевод на тот же счёт.
type sameAccount struct{}

func (sameAccount) Code() string { return domain.RULE_SAME_ACCOUNT }

func (sameAccount) Check(_ context.Context, transfer models.TransferCheck, _ interfaces.CheckStats) (string, error) {
	if transfer.FromAccountID != transfer.ToAccountID {
		return "", nil
	}

	return "source and destination accounts are the same", nil
}
