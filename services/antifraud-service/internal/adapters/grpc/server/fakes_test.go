package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

const (
	TEST_INITIATOR_ID    = "11111111-1111-1111-1111-111111111111"
	TEST_FROM_ACCOUNT_ID = "22222222-2222-2222-2222-222222222222"
	TEST_TO_ACCOUNT_ID   = "33333333-3333-3333-3333-333333333333"
	TEST_CHECK_ID        = "44444444-4444-4444-4444-444444444444"
	TEST_OFFICER_ID      = "55555555-5555-5555-5555-555555555555"
	TEST_AMOUNT          = int64(1500)
	TEST_SECRET          = "secret-dsn"
)

// errDB - сбой базы, текст которого не должен уйти наружу
var errDB = errors.New("db: connection refused " + TEST_SECRET)

// testUpdatedAt - фиксированный момент правки правила
func testUpdatedAt() time.Time {
	return time.Date(2026, 9, 28, 15, 30, 0, 0, time.UTC)
}

// fakeCheckService отдаёт заданную проверку и записывает аргументы
type fakeCheckService struct {
	check *models.Check
	err   error
	calls []models.TransferCheck
}

var _ interfaces.CheckService = (*fakeCheckService)(nil)

func (f *fakeCheckService) CheckTransfer(_ context.Context, transfer models.TransferCheck) (*models.Check, error) {
	f.calls = append(f.calls, transfer)
	if f.err != nil {
		return nil, f.err
	}
	return f.check, nil
}

// fakeRuleService отдаёт заданные правила и записывает аргументы
type fakeRuleService struct {
	listResult   []models.RuleConfig
	listErr      error
	listCalls    int
	updateResult *models.RuleConfig
	updateErr    error
	updateCalls  []models.RuleUpdate
}

var _ interfaces.RuleService = (*fakeRuleService)(nil)

func (f *fakeRuleService) ListRules(_ context.Context) ([]models.RuleConfig, error) {
	f.listCalls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResult, nil
}

func (f *fakeRuleService) UpdateRule(_ context.Context, update models.RuleUpdate) (*models.RuleConfig, error) {
	f.updateCalls = append(f.updateCalls, update)
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return f.updateResult, nil
}

// newTestServer собирает сервер с фейками и немым логгером
func newTestServer(checks *fakeCheckService, rules *fakeRuleService) *AntifraudServer {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewAntifraudServer(&service.Service{Checks: checks, Rules: rules}, log)
}

// boolPtr - указатель на значение bool
func boolPtr(value bool) *bool {
	return &value
}

// uuidPtr - указатель на разобранный uuid
func uuidPtr(raw string) *uuid.UUID {
	parsed := uuid.MustParse(raw)
	return &parsed
}

// normalizeEmptyLimits приводит пустые Limits к nil: nil и пустая map для R10 равнозначны
func normalizeEmptyLimits(params *models.RuleParams) *models.RuleParams {
	if params == nil {
		return nil
	}
	copied := *params
	if len(copied.Limits) == 0 {
		copied.Limits = nil
	}
	return &copied
}

// normalizeUpdates применяет normalizeEmptyLimits к записанным вызовам UpdateRule
func normalizeUpdates(updates []models.RuleUpdate) []models.RuleUpdate {
	normalized := make([]models.RuleUpdate, 0, len(updates))
	for _, update := range updates {
		update.Params = normalizeEmptyLimits(update.Params)
		normalized = append(normalized, update)
	}
	return normalized
}
