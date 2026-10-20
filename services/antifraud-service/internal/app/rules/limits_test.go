package rules

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

// Фиксированные идентификаторы, чтобы сверять аргументы вызовов статистики.
var (
	TEST_INITIATOR_ID    = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	TEST_FROM_ACCOUNT_ID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	TEST_TO_ACCOUNT_ID   = uuid.MustParse("33333333-3333-3333-3333-333333333333")
)

const (
	TEST_RUB models.Currency = "RUB"
	TEST_USD models.Currency = "USD"
	TEST_EUR models.Currency = "EUR"

	TEST_LIMIT int64 = 1000
)

// fixedTransfer - перевод с фиксированными счетами в момент testNow.
func fixedTransfer(amount int64, currency models.Currency) models.TransferCheck {
	return models.TransferCheck{
		InitiatorID:   TEST_INITIATOR_ID,
		FromAccountID: TEST_FROM_ACCOUNT_ID,
		ToAccountID:   TEST_TO_ACCOUNT_ID,
		Amount:        amount,
		Currency:      currency,
		At:            testNow,
	}
}

func TestSingleAmountLimitCheck(t *testing.T) {
	tests := []struct {
		name     string
		limits   map[models.Currency]int64
		transfer models.TransferCheck
		wantHit  bool
	}{
		{"below limit", map[models.Currency]int64{TEST_RUB: TEST_LIMIT}, fixedTransfer(999, TEST_RUB), false},
		{"exactly limit", map[models.Currency]int64{TEST_RUB: TEST_LIMIT}, fixedTransfer(1000, TEST_RUB), false},
		{"one above limit", map[models.Currency]int64{TEST_RUB: TEST_LIMIT}, fixedTransfer(1001, TEST_RUB), true},
		{"usd without limit", map[models.Currency]int64{TEST_RUB: TEST_LIMIT}, fixedTransfer(1_000_000, TEST_USD), false},
		{"eur without limit", map[models.Currency]int64{TEST_RUB: TEST_LIMIT}, fixedTransfer(1, TEST_EUR), false},
		{"nil limits", nil, fixedTransfer(1_000_000, TEST_RUB), false},
		{"minimal limit exceeded", map[models.Currency]int64{TEST_RUB: 1}, fixedTransfer(2, TEST_RUB), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := singleAmountLimit{limits: tt.limits}

			// stats nil: правило в статистику не ходит
			reason, err := rule.Check(t.Context(), tt.transfer, nil)

			require.NoError(t, err)
			assert.Equal(t, tt.wantHit, reason != "")
		})
	}
}

func TestDailyAmountLimitCheck(t *testing.T) {
	testDay := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	// 02:30+05:00 28-го = 21:30Z 27-го
	eastAt := time.Date(2026, 9, 28, 2, 30, 0, 0, time.FixedZone("UTC+5", 5*3600))
	eastTransfer := fixedTransfer(1, TEST_RUB)
	eastTransfer.At = eastAt

	tests := []struct {
		name       string
		limits     map[models.Currency]int64
		transfer   models.TransferCheck
		dailyTotal int64
		dailyErr   error
		wantHit    bool
		wantCalls  []dailyTotalCall
	}{
		{
			name: "total plus amount equals limit", limits: map[models.Currency]int64{TEST_RUB: TEST_LIMIT},
			transfer: fixedTransfer(600, TEST_RUB), dailyTotal: 400, wantHit: false,
			wantCalls: []dailyTotalCall{{AccountID: TEST_FROM_ACCOUNT_ID, Currency: TEST_RUB, Day: testDay}},
		},
		{
			name: "total plus amount one above limit", limits: map[models.Currency]int64{TEST_RUB: TEST_LIMIT},
			transfer: fixedTransfer(501, TEST_RUB), dailyTotal: 500, wantHit: true,
			wantCalls: []dailyTotalCall{{AccountID: TEST_FROM_ACCOUNT_ID, Currency: TEST_RUB, Day: testDay}},
		},
		{
			name: "total plus amount one below limit", limits: map[models.Currency]int64{TEST_RUB: TEST_LIMIT},
			transfer: fixedTransfer(599, TEST_RUB), dailyTotal: 400, wantHit: false,
			wantCalls: []dailyTotalCall{{AccountID: TEST_FROM_ACCOUNT_ID, Currency: TEST_RUB, Day: testDay}},
		},
		{
			name: "limit already reached", limits: map[models.Currency]int64{TEST_RUB: TEST_LIMIT},
			transfer: fixedTransfer(1, TEST_RUB), dailyTotal: 1000, wantHit: true,
			wantCalls: []dailyTotalCall{{AccountID: TEST_FROM_ACCOUNT_ID, Currency: TEST_RUB, Day: testDay}},
		},
		{
			name: "empty day exactly limit", limits: map[models.Currency]int64{TEST_RUB: TEST_LIMIT},
			transfer: fixedTransfer(1000, TEST_RUB), dailyTotal: 0, wantHit: false,
			wantCalls: []dailyTotalCall{{AccountID: TEST_FROM_ACCOUNT_ID, Currency: TEST_RUB, Day: testDay}},
		},
		{
			name: "empty day single transfer above limit", limits: map[models.Currency]int64{TEST_RUB: TEST_LIMIT},
			transfer: fixedTransfer(1001, TEST_RUB), dailyTotal: 0, wantHit: true,
			wantCalls: []dailyTotalCall{{AccountID: TEST_FROM_ACCOUNT_ID, Currency: TEST_RUB, Day: testDay}},
		},
		{
			name: "non utc time maps to utc day", limits: map[models.Currency]int64{TEST_RUB: TEST_LIMIT},
			transfer: eastTransfer, dailyTotal: 0, wantHit: false,
			wantCalls: []dailyTotalCall{{AccountID: TEST_FROM_ACCOUNT_ID, Currency: TEST_RUB, Day: time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)}},
		},
		{
			// ошибка статистики не должна всплыть: в неё не ходим
			name: "currency without limit skips stats", limits: map[models.Currency]int64{TEST_RUB: TEST_LIMIT},
			transfer: fixedTransfer(1_000_000, TEST_USD), dailyErr: errors.New("db failure"), wantHit: false,
			wantCalls: nil,
		},
		{
			name: "nil limits skips stats", limits: nil,
			transfer: fixedTransfer(1_000_000, TEST_RUB), dailyErr: errors.New("db failure"), wantHit: false,
			wantCalls: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := dailyAmountLimit{limits: tt.limits}
			stats := &recordingStats{dailyTotal: tt.dailyTotal, dailyErr: tt.dailyErr}

			reason, err := rule.Check(t.Context(), tt.transfer, stats)

			require.NoError(t, err)
			assert.Equal(t, tt.wantHit, reason != "")
			assert.Equal(t, tt.wantCalls, stats.dailyCalls)
			assert.Nil(t, stats.countCalls)
		})
	}
}

func TestDailyAmountLimitStatsError(t *testing.T) {
	errDB := errors.New("db failure")
	rule := dailyAmountLimit{limits: map[models.Currency]int64{TEST_RUB: TEST_LIMIT}}
	stats := &recordingStats{dailyErr: errDB}

	reason, err := rule.Check(t.Context(), fixedTransfer(1, TEST_RUB), stats)

	require.ErrorIs(t, err, errDB)
	assert.Equal(t, "", reason)
}

func TestVelocityCheck(t *testing.T) {
	tests := []struct {
		name      string
		params    models.RuleParams
		allowed   int
		wantHit   bool
		wantSince time.Time
	}{
		{"first transfer", models.RuleParams{MaxCount: 3, WindowSeconds: 60}, 0, false, time.Date(2026, 9, 28, 15, 29, 0, 0, time.UTC)},
		{"last allowed", models.RuleParams{MaxCount: 3, WindowSeconds: 60}, 2, false, time.Date(2026, 9, 28, 15, 29, 0, 0, time.UTC)},
		{"count equals max", models.RuleParams{MaxCount: 3, WindowSeconds: 60}, 3, true, time.Date(2026, 9, 28, 15, 29, 0, 0, time.UTC)},
		{"count far above max", models.RuleParams{MaxCount: 3, WindowSeconds: 60}, 10, true, time.Date(2026, 9, 28, 15, 29, 0, 0, time.UTC)},
		{"hour window", models.RuleParams{MaxCount: 3, WindowSeconds: 3600}, 0, false, time.Date(2026, 9, 28, 14, 30, 0, 0, time.UTC)},
		{"max one with no transfers", models.RuleParams{MaxCount: 1, WindowSeconds: 60}, 0, false, time.Date(2026, 9, 28, 15, 29, 0, 0, time.UTC)},
		{"max one with one transfer", models.RuleParams{MaxCount: 1, WindowSeconds: 60}, 1, true, time.Date(2026, 9, 28, 15, 29, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := newVelocity(tt.params)
			stats := &recordingStats{allowed: tt.allowed}

			reason, err := rule.Check(t.Context(), fixedTransfer(100, TEST_RUB), stats)

			require.NoError(t, err)
			assert.Equal(t, tt.wantHit, reason != "")
			assert.Equal(t, []countAllowedCall{{AccountID: TEST_FROM_ACCOUNT_ID, Since: tt.wantSince}}, stats.countCalls)
			assert.Nil(t, stats.dailyCalls)
		})
	}
}

func TestVelocityStatsError(t *testing.T) {
	errDB := errors.New("db failure")
	rule := newVelocity(models.RuleParams{MaxCount: 3, WindowSeconds: 60})
	stats := &recordingStats{countErr: errDB}

	reason, err := rule.Check(t.Context(), fixedTransfer(100, TEST_RUB), stats)

	require.ErrorIs(t, err, errDB)
	assert.Equal(t, "", reason)
}

func TestSameAccountCheck(t *testing.T) {
	sameTransfer := fixedTransfer(100, TEST_RUB)
	sameTransfer.ToAccountID = TEST_FROM_ACCOUNT_ID

	tests := []struct {
		name     string
		transfer models.TransferCheck
		wantHit  bool
	}{
		{"different accounts", fixedTransfer(100, TEST_RUB), false},
		{"same account", sameTransfer, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// stats nil: правило в статистику не ходит
			reason, err := (sameAccount{}).Check(t.Context(), tt.transfer, nil)

			require.NoError(t, err)
			assert.Equal(t, tt.wantHit, reason != "")
		})
	}
}

func TestRuleCodes(t *testing.T) {
	tests := []struct {
		name string
		rule Rule
		want string
	}{
		{"single amount limit", singleAmountLimit{}, domain.RULE_SINGLE_AMOUNT_LIMIT},
		{"daily amount limit", dailyAmountLimit{}, domain.RULE_DAILY_AMOUNT_LIMIT},
		{"velocity", newVelocity(models.RuleParams{MaxCount: 1, WindowSeconds: 60}), domain.RULE_VELOCITY},
		{"same account", sameAccount{}, domain.RULE_SAME_ACCOUNT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.rule.Code())
		})
	}
}
