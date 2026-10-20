package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

// errDB - ошибка базы для режимов отказа фейков
var errDB = errors.New("db failure")

var (
	TEST_INITIATOR_ID    = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	TEST_FROM_ACCOUNT_ID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	TEST_TO_ACCOUNT_ID   = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	TEST_CHECK_ID        = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	TEST_OFFICER_ID      = uuid.MustParse("55555555-5555-5555-5555-555555555555")
)

var _ interfaces.RuleRepository = (*fakeRuleRepo)(nil)

// fakeRuleRepo отдаёт заданные ответы и записывает вызовы
type fakeRuleRepo struct {
	listQueue    [][]models.RuleConfig
	listErr      error
	listCalls    int
	updateResult *models.RuleConfig
	updateErr    error
	updateCalls  []models.RuleUpdate
}

func (f *fakeRuleRepo) ListRules(_ context.Context) ([]models.RuleConfig, error) {
	f.listCalls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	if len(f.listQueue) == 0 {
		return nil, nil
	}
	// очередь короче числа вызовов - повторяем последний ответ
	idx := min(f.listCalls-1, len(f.listQueue)-1)
	return f.listQueue[idx], nil
}

func (f *fakeRuleRepo) UpdateRule(_ context.Context, update models.RuleUpdate) (*models.RuleConfig, error) {
	f.updateCalls = append(f.updateCalls, update)
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return f.updateResult, nil
}

var _ interfaces.CheckStats = (*fakeStats)(nil)

type countAllowedCall struct {
	accountID uuid.UUID
	since     time.Time
}

type dailyTotalCall struct {
	accountID uuid.UUID
	currency  models.Currency
	day       time.Time
}

// fakeStats отдаёт заданную статистику и записывает аргументы
type fakeStats struct {
	allowed    int
	allowedErr error
	total      int64
	totalErr   error
	countCalls []countAllowedCall
	dailyCalls []dailyTotalCall
}

func (f *fakeStats) CountAllowed(_ context.Context, accountID uuid.UUID, since time.Time) (int, error) {
	f.countCalls = append(f.countCalls, countAllowedCall{accountID: accountID, since: since})
	if f.allowedErr != nil {
		return 0, f.allowedErr
	}
	return f.allowed, nil
}

func (f *fakeStats) DailyTotal(_ context.Context, accountID uuid.UUID, currency models.Currency, day time.Time) (int64, error) {
	f.dailyCalls = append(f.dailyCalls, dailyTotalCall{accountID: accountID, currency: currency, day: day})
	if f.totalErr != nil {
		return 0, f.totalErr
	}
	return f.total, nil
}

var _ interfaces.CheckRepository = (*fakeCheckRepo)(nil)

// fakeCheckRepo по контракту Record вызывает evaluate со статистикой и собирает запись
type fakeCheckRepo struct {
	checkID    uuid.UUID
	stats      interfaces.CheckStats
	recordErr  error
	transfers  []models.TransferCheck
	lastHits   []models.Hit
	lastResult *models.Check
}

func (f *fakeCheckRepo) Record(ctx context.Context, transfer models.TransferCheck, evaluate interfaces.EvaluateFunc) (*models.Check, error) {
	f.transfers = append(f.transfers, transfer)
	if f.recordErr != nil {
		return nil, f.recordErr
	}
	hits, err := evaluate(ctx, f.stats)
	if err != nil {
		return nil, fmt.Errorf("evaluate: %w", err)
	}
	f.lastHits = hits
	f.lastResult = &models.Check{
		ID:       f.checkID,
		Transfer: transfer,
		Verdict:  models.VerdictFor(hits),
		Hits:     hits,
	}
	return f.lastResult, nil
}
