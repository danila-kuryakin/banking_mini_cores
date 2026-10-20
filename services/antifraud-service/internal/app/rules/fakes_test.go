package rules

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

// countAllowedCall - аргументы одного вызова CountAllowed.
type countAllowedCall struct {
	AccountID uuid.UUID
	Since     time.Time
}

// dailyTotalCall - аргументы одного вызова DailyTotal.
type dailyTotalCall struct {
	AccountID uuid.UUID
	Currency  models.Currency
	Day       time.Time
}

// recordingStats - статистика счёта с заданными ошибками и записью вызовов.
type recordingStats struct {
	allowed    int
	countErr   error
	dailyTotal int64
	dailyErr   error

	countCalls []countAllowedCall
	dailyCalls []dailyTotalCall
}

var _ interfaces.CheckStats = (*recordingStats)(nil)

func (s *recordingStats) CountAllowed(_ context.Context, accountID uuid.UUID, since time.Time) (int, error) {
	s.countCalls = append(s.countCalls, countAllowedCall{AccountID: accountID, Since: since})
	if s.countErr != nil {
		return 0, s.countErr
	}

	return s.allowed, nil
}

func (s *recordingStats) DailyTotal(_ context.Context, accountID uuid.UUID, currency models.Currency, day time.Time) (int64, error) {
	s.dailyCalls = append(s.dailyCalls, dailyTotalCall{AccountID: accountID, Currency: currency, Day: day})
	if s.dailyErr != nil {
		return 0, s.dailyErr
	}

	return s.dailyTotal, nil
}

// stubRule - правило с заданным кодом, причиной и ошибкой, считает вызовы.
type stubRule struct {
	code   string
	reason string
	err    error

	calls     int
	transfers []models.TransferCheck
	stats     []interfaces.CheckStats
}

var _ Rule = (*stubRule)(nil)

func (r *stubRule) Code() string {
	return r.code
}

func (r *stubRule) Check(_ context.Context, transfer models.TransferCheck, stats interfaces.CheckStats) (string, error) {
	r.calls++
	r.transfers = append(r.transfers, transfer)
	r.stats = append(r.stats, stats)
	if r.err != nil {
		return "", r.err
	}

	return r.reason, nil
}
