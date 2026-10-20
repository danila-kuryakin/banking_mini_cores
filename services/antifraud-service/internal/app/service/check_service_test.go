package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

const (
	TEST_RUB    models.Currency = "RUB"
	TEST_USD    models.Currency = "USD"
	TEST_WINDOW                 = 60 * time.Second
)

// TEST_OLD_AT - время во входе, которое сервис обязан игнорировать
var TEST_OLD_AT = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

func singleLimitRule(enabled bool) models.RuleConfig {
	return models.RuleConfig{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Enabled: enabled, Params: models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 1000}}}
}

func dailyLimitRule(enabled bool) models.RuleConfig {
	return models.RuleConfig{Code: domain.RULE_DAILY_AMOUNT_LIMIT, Enabled: enabled, Params: models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 1000}}}
}

func velocityRule(enabled bool) models.RuleConfig {
	return models.RuleConfig{Code: domain.RULE_VELOCITY, Enabled: enabled, Params: models.RuleParams{MaxCount: 3, WindowSeconds: 60}}
}

func sameAccountRule(enabled bool) models.RuleConfig {
	return models.RuleConfig{Code: domain.RULE_SAME_ACCOUNT, Enabled: enabled}
}

func newTransfer(toAccountID uuid.UUID, amount int64, currency models.Currency) models.TransferCheck {
	return models.TransferCheck{
		InitiatorID:   TEST_INITIATOR_ID,
		FromAccountID: TEST_FROM_ACCOUNT_ID,
		ToAccountID:   toAccountID,
		Amount:        amount,
		Currency:      currency,
		At:            TEST_OLD_AT,
	}
}

func hitCodes(hits []models.Hit) []string {
	codes := make([]string, 0, len(hits))
	for _, hit := range hits {
		codes = append(codes, hit.RuleCode)
	}
	return codes
}

func hitReasons(hits []models.Hit) []string {
	reasons := make([]string, 0, len(hits))
	for _, hit := range hits {
		reasons = append(reasons, hit.Reason)
	}
	return reasons
}

func noCountCalls(time.Time) []countAllowedCall { return nil }

func noDailyCalls(time.Time) []dailyTotalCall { return nil }

func velocityCalls(at time.Time) []countAllowedCall {
	return []countAllowedCall{{accountID: TEST_FROM_ACCOUNT_ID, since: at.Add(-TEST_WINDOW)}}
}

func dailyRUBCalls(at time.Time) []dailyTotalCall {
	return []dailyTotalCall{{accountID: TEST_FROM_ACCOUNT_ID, currency: TEST_RUB, day: models.Day(at)}}
}

func TestCheckTransfer_DependencyErrors(t *testing.T) {
	tests := []struct {
		name            string
		listErr         error
		recordErr       error
		rules           []models.RuleConfig
		stats           fakeStats
		wantRecordCalls int
		wantStatsCalls  int
	}{
		{name: "list rules error", listErr: errDB, wantRecordCalls: 0, wantStatsCalls: 0},
		{name: "record error", recordErr: errDB, rules: []models.RuleConfig{velocityRule(true)}, wantRecordCalls: 1, wantStatsCalls: 0},
		{name: "count allowed error", rules: []models.RuleConfig{velocityRule(true)}, stats: fakeStats{allowedErr: errDB}, wantRecordCalls: 1, wantStatsCalls: 1},
		{name: "daily total error", rules: []models.RuleConfig{dailyLimitRule(true)}, stats: fakeStats{totalErr: errDB}, wantRecordCalls: 1, wantStatsCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := tt.stats
			ruleRepo := &fakeRuleRepo{listQueue: [][]models.RuleConfig{tt.rules}, listErr: tt.listErr}
			checkRepo := &fakeCheckRepo{checkID: TEST_CHECK_ID, stats: &stats, recordErr: tt.recordErr}
			svc := NewCheckService(checkRepo, ruleRepo)

			got, err := svc.CheckTransfer(t.Context(), newTransfer(TEST_TO_ACCOUNT_ID, 500, TEST_RUB))

			require.ErrorIs(t, err, errDB)
			assert.Nil(t, got)
			assert.Len(t, checkRepo.transfers, tt.wantRecordCalls)
			assert.Equal(t, tt.wantStatsCalls, len(stats.countCalls)+len(stats.dailyCalls))
		})
	}
}

func TestCheckTransfer_UnknownEnabledRule(t *testing.T) {
	tests := []struct {
		name  string
		rules []models.RuleConfig
	}{
		{name: "only unknown enabled", rules: []models.RuleConfig{{Code: "unknown", Enabled: true}}},
		{name: "unknown enabled after valid rule", rules: []models.RuleConfig{singleLimitRule(true), {Code: "unknown", Enabled: true}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleRepo := &fakeRuleRepo{listQueue: [][]models.RuleConfig{tt.rules}}
			checkRepo := &fakeCheckRepo{checkID: TEST_CHECK_ID, stats: &fakeStats{}}
			svc := NewCheckService(checkRepo, ruleRepo)

			got, err := svc.CheckTransfer(t.Context(), newTransfer(TEST_TO_ACCOUNT_ID, 500, TEST_RUB))

			require.Error(t, err)
			assert.Nil(t, got)
			assert.Empty(t, checkRepo.transfers)
		})
	}
}

func TestCheckTransfer_RecordedTransfer(t *testing.T) {
	tests := []struct {
		name  string
		input models.TransferCheck
	}{
		{name: "old at is replaced", input: newTransfer(TEST_TO_ACCOUNT_ID, 700, TEST_RUB)},
		{name: "zero at is replaced", input: models.TransferCheck{InitiatorID: TEST_INITIATOR_ID, FromAccountID: TEST_FROM_ACCOUNT_ID, ToAccountID: TEST_TO_ACCOUNT_ID, Amount: 1, Currency: TEST_USD}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleRepo := &fakeRuleRepo{}
			checkRepo := &fakeCheckRepo{checkID: TEST_CHECK_ID, stats: &fakeStats{}}
			svc := NewCheckService(checkRepo, ruleRepo)

			before := time.Now()
			_, err := svc.CheckTransfer(t.Context(), tt.input)
			after := time.Now()

			require.NoError(t, err)
			require.Len(t, checkRepo.transfers, 1)
			recorded := checkRepo.transfers[0]
			assert.Equal(t, time.UTC, recorded.At.Location())
			assert.False(t, recorded.At.Before(before.Truncate(0)), "at %v is before %v", recorded.At, before)
			assert.False(t, recorded.At.After(after.Truncate(0)), "at %v is after %v", recorded.At, after)
			// кроме At, перевод совпадает со входом
			want := tt.input
			want.At = recorded.At
			assert.Equal(t, want, recorded)
		})
	}
}

func TestCheckTransfer_ReturnsRecordResult(t *testing.T) {
	ruleRepo := &fakeRuleRepo{listQueue: [][]models.RuleConfig{{singleLimitRule(true)}}}
	checkRepo := &fakeCheckRepo{checkID: TEST_CHECK_ID, stats: &fakeStats{}}
	svc := NewCheckService(checkRepo, ruleRepo)

	got, err := svc.CheckTransfer(t.Context(), newTransfer(TEST_TO_ACCOUNT_ID, 1500, TEST_RUB))

	require.NoError(t, err)
	require.NotNil(t, checkRepo.lastResult)
	assert.Same(t, checkRepo.lastResult, got)
	assert.Equal(t, TEST_CHECK_ID, got.ID)
}

func TestCheckTransfer_Evaluate(t *testing.T) {
	tests := []struct {
		name        string
		rules       []models.RuleConfig
		toAccountID uuid.UUID
		amount      int64
		currency    models.Currency
		allowed     int
		total       int64
		wantCodes   []string
		wantVerdict models.Verdict
		wantCount   func(at time.Time) []countAllowedCall
		wantDaily   func(at time.Time) []dailyTotalCall
	}{
		{name: "disabled unknown rule is ignored", rules: []models.RuleConfig{{Code: "unknown", Enabled: false}}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 1500, currency: TEST_RUB, wantCodes: []string{}, wantVerdict: models.VERDICT_ALLOW, wantCount: noCountCalls, wantDaily: noDailyCalls},
		{name: "single limit exceeded", rules: []models.RuleConfig{singleLimitRule(true)}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 1500, currency: TEST_RUB, wantCodes: []string{domain.RULE_SINGLE_AMOUNT_LIMIT}, wantVerdict: models.VERDICT_DENY, wantCount: noCountCalls, wantDaily: noDailyCalls},
		{name: "single limit equal amount", rules: []models.RuleConfig{singleLimitRule(true)}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 1000, currency: TEST_RUB, wantCodes: []string{}, wantVerdict: models.VERDICT_ALLOW, wantCount: noCountCalls, wantDaily: noDailyCalls},
		{name: "single limit plus one", rules: []models.RuleConfig{singleLimitRule(true)}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 1001, currency: TEST_RUB, wantCodes: []string{domain.RULE_SINGLE_AMOUNT_LIMIT}, wantVerdict: models.VERDICT_DENY, wantCount: noCountCalls, wantDaily: noDailyCalls},
		{name: "single limit currency without threshold", rules: []models.RuleConfig{singleLimitRule(true)}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 5000, currency: TEST_USD, wantCodes: []string{}, wantVerdict: models.VERDICT_ALLOW, wantCount: noCountCalls, wantDaily: noDailyCalls},
		{name: "velocity count equals max", rules: []models.RuleConfig{velocityRule(true)}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 10, currency: TEST_RUB, allowed: 3, wantCodes: []string{domain.RULE_VELOCITY}, wantVerdict: models.VERDICT_DENY, wantCount: velocityCalls, wantDaily: noDailyCalls},
		{name: "velocity count below max", rules: []models.RuleConfig{velocityRule(true)}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 10, currency: TEST_RUB, allowed: 2, wantCodes: []string{}, wantVerdict: models.VERDICT_ALLOW, wantCount: velocityCalls, wantDaily: noDailyCalls},
		{name: "velocity count above max", rules: []models.RuleConfig{velocityRule(true)}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 10, currency: TEST_RUB, allowed: 4, wantCodes: []string{domain.RULE_VELOCITY}, wantVerdict: models.VERDICT_DENY, wantCount: velocityCalls, wantDaily: noDailyCalls},
		// 600 + 500 = 1100 > 1000
		{name: "daily limit exceeded", rules: []models.RuleConfig{dailyLimitRule(true)}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 500, currency: TEST_RUB, total: 600, wantCodes: []string{domain.RULE_DAILY_AMOUNT_LIMIT}, wantVerdict: models.VERDICT_DENY, wantCount: noCountCalls, wantDaily: dailyRUBCalls},
		// 500 + 500 = 1000, не больше порога
		{name: "daily limit reached exactly", rules: []models.RuleConfig{dailyLimitRule(true)}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 500, currency: TEST_RUB, total: 500, wantCodes: []string{}, wantVerdict: models.VERDICT_ALLOW, wantCount: noCountCalls, wantDaily: dailyRUBCalls},
		{name: "daily limit currency without threshold", rules: []models.RuleConfig{dailyLimitRule(true)}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 5000, currency: TEST_USD, total: 600, wantCodes: []string{}, wantVerdict: models.VERDICT_ALLOW, wantCount: noCountCalls, wantDaily: noDailyCalls},
		{name: "same account", rules: []models.RuleConfig{sameAccountRule(true)}, toAccountID: TEST_FROM_ACCOUNT_ID, amount: 10, currency: TEST_RUB, wantCodes: []string{domain.RULE_SAME_ACCOUNT}, wantVerdict: models.VERDICT_DENY, wantCount: noCountCalls, wantDaily: noDailyCalls},
		{name: "different accounts", rules: []models.RuleConfig{sameAccountRule(true)}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 10, currency: TEST_RUB, wantCodes: []string{}, wantVerdict: models.VERDICT_ALLOW, wantCount: noCountCalls, wantDaily: noDailyCalls},
		{name: "all rules enabled hits in config order", rules: []models.RuleConfig{singleLimitRule(true), velocityRule(true), dailyLimitRule(true), sameAccountRule(true)}, toAccountID: TEST_FROM_ACCOUNT_ID, amount: 1500, currency: TEST_RUB, wantCodes: []string{domain.RULE_SINGLE_AMOUNT_LIMIT, domain.RULE_DAILY_AMOUNT_LIMIT, domain.RULE_SAME_ACCOUNT}, wantVerdict: models.VERDICT_DENY, wantCount: velocityCalls, wantDaily: dailyRUBCalls},
		{name: "all rules disabled", rules: []models.RuleConfig{singleLimitRule(false), velocityRule(false), dailyLimitRule(false), sameAccountRule(false)}, toAccountID: TEST_FROM_ACCOUNT_ID, amount: 1500, currency: TEST_RUB, allowed: 10, total: 10000, wantCodes: []string{}, wantVerdict: models.VERDICT_ALLOW, wantCount: noCountCalls, wantDaily: noDailyCalls},
		{name: "nil rules", rules: nil, toAccountID: TEST_FROM_ACCOUNT_ID, amount: 1500, currency: TEST_RUB, wantCodes: []string{}, wantVerdict: models.VERDICT_ALLOW, wantCount: noCountCalls, wantDaily: noDailyCalls},
		{name: "empty rules", rules: []models.RuleConfig{}, toAccountID: TEST_FROM_ACCOUNT_ID, amount: 1500, currency: TEST_RUB, wantCodes: []string{}, wantVerdict: models.VERDICT_ALLOW, wantCount: noCountCalls, wantDaily: noDailyCalls},
		{name: "disabled limit is not applied", rules: []models.RuleConfig{singleLimitRule(false), sameAccountRule(true)}, toAccountID: TEST_TO_ACCOUNT_ID, amount: 1500, currency: TEST_RUB, wantCodes: []string{}, wantVerdict: models.VERDICT_ALLOW, wantCount: noCountCalls, wantDaily: noDailyCalls},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &fakeStats{allowed: tt.allowed, total: tt.total}
			ruleRepo := &fakeRuleRepo{listQueue: [][]models.RuleConfig{tt.rules}}
			checkRepo := &fakeCheckRepo{checkID: TEST_CHECK_ID, stats: stats}
			svc := NewCheckService(checkRepo, ruleRepo)

			got, err := svc.CheckTransfer(t.Context(), newTransfer(tt.toAccountID, tt.amount, tt.currency))

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Len(t, checkRepo.transfers, 1)
			at := checkRepo.transfers[0].At
			assert.Equal(t, tt.wantCodes, hitCodes(got.Hits))
			assert.NotContains(t, hitReasons(got.Hits), "")
			assert.Equal(t, tt.wantVerdict, got.Verdict)
			assert.Equal(t, tt.wantCount(at), stats.countCalls)
			assert.Equal(t, tt.wantDaily(at), stats.dailyCalls)
		})
	}
}

func TestCheckTransfer_ReadsRulesEachTime(t *testing.T) {
	ruleRepo := &fakeRuleRepo{listQueue: [][]models.RuleConfig{
		{singleLimitRule(false)},
		{singleLimitRule(true)},
	}}
	checkRepo := &fakeCheckRepo{checkID: TEST_CHECK_ID, stats: &fakeStats{}}
	svc := NewCheckService(checkRepo, ruleRepo)
	transfer := newTransfer(TEST_TO_ACCOUNT_ID, 1500, TEST_RUB)

	first, err := svc.CheckTransfer(t.Context(), transfer)
	require.NoError(t, err)
	second, err := svc.CheckTransfer(t.Context(), transfer)
	require.NoError(t, err)

	assert.Equal(t, 2, ruleRepo.listCalls)
	assert.Len(t, checkRepo.transfers, 2)
	assert.Equal(t, models.VERDICT_ALLOW, first.Verdict)
	assert.Equal(t, models.VERDICT_DENY, second.Verdict)
}

func TestCheckTransfer_Invariants(t *testing.T) {
	tests := []struct {
		name  string
		rules []models.RuleConfig
		stats fakeStats
		input models.TransferCheck
	}{
		{name: "no rules", input: newTransfer(TEST_TO_ACCOUNT_ID, 100, TEST_RUB)},
		{name: "single limit hit", rules: []models.RuleConfig{singleLimitRule(true)}, input: newTransfer(TEST_TO_ACCOUNT_ID, 1500, TEST_RUB)},
		{name: "velocity hit", rules: []models.RuleConfig{velocityRule(true)}, stats: fakeStats{allowed: 5}, input: newTransfer(TEST_TO_ACCOUNT_ID, 10, TEST_USD)},
		{name: "all rules same account", rules: []models.RuleConfig{singleLimitRule(true), velocityRule(true), dailyLimitRule(true), sameAccountRule(true)}, input: newTransfer(TEST_FROM_ACCOUNT_ID, 2000, TEST_RUB)},
		{name: "disabled same account and daily below limit", rules: []models.RuleConfig{sameAccountRule(false), dailyLimitRule(true)}, stats: fakeStats{total: 100}, input: newTransfer(TEST_FROM_ACCOUNT_ID, 50, TEST_RUB)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := tt.stats
			ruleRepo := &fakeRuleRepo{listQueue: [][]models.RuleConfig{tt.rules}}
			checkRepo := &fakeCheckRepo{checkID: TEST_CHECK_ID, stats: &stats}
			svc := NewCheckService(checkRepo, ruleRepo)

			got, err := svc.CheckTransfer(t.Context(), tt.input)

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Len(t, checkRepo.transfers, 1)
			recorded := checkRepo.transfers[0]
			want := tt.input
			want.At = recorded.At
			assert.Equal(t, want, recorded)
			assert.Equal(t, recorded, got.Transfer)
			assert.Equal(t, checkRepo.lastHits, got.Hits)
			assert.Equal(t, models.VerdictFor(got.Hits), got.Verdict)
		})
	}
}
