package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/pb/gen/antifraud/v1"
)

// validCheckRequest - корректный запрос проверки с заданной суммой и валютой
func validCheckRequest(amount int64, currency string) *antifraudv1.CheckTransferRequest {
	return &antifraudv1.CheckTransferRequest{
		InitiatorId:   TEST_INITIATOR_ID,
		FromAccountId: TEST_FROM_ACCOUNT_ID,
		ToAccountId:   TEST_TO_ACCOUNT_ID,
		Amount:        amount,
		Currency:      currency,
	}
}

// expectedTransfer - что сервис должен получить для validCheckRequest
func expectedTransfer(amount int64, currency models.Currency) models.TransferCheck {
	return models.TransferCheck{
		InitiatorID:   uuid.MustParse(TEST_INITIATOR_ID),
		FromAccountID: uuid.MustParse(TEST_FROM_ACCOUNT_ID),
		ToAccountID:   uuid.MustParse(TEST_TO_ACCOUNT_ID),
		Amount:        amount,
		Currency:      currency,
	}
}

func TestCheckTransfer_Success(t *testing.T) {
	checkID := uuid.MustParse(TEST_CHECK_ID)

	tests := []struct {
		name      string
		req       *antifraudv1.CheckTransferRequest
		check     *models.Check
		wantCalls []models.TransferCheck
		want      *antifraudv1.CheckTransferResponse
	}{
		{
			name: "deny with two hits keeps order and normalizes currency",
			req:  validCheckRequest(TEST_AMOUNT, "rub"),
			check: &models.Check{ID: checkID, Verdict: models.VERDICT_DENY, Hits: []models.Hit{
				{RuleCode: "velocity", Reason: "3 transfers"},
				{RuleCode: "same_account", Reason: "same"},
			}},
			wantCalls: []models.TransferCheck{expectedTransfer(TEST_AMOUNT, "RUB")},
			want: &antifraudv1.CheckTransferResponse{
				CheckId: TEST_CHECK_ID,
				Verdict: antifraudv1.Verdict_VERDICT_DENY,
				Reasons: []string{"velocity: 3 transfers", "same_account: same"},
			},
		},
		{
			name:      "allow with nil hits",
			req:       validCheckRequest(TEST_AMOUNT, "RUB"),
			check:     &models.Check{ID: checkID, Verdict: models.VERDICT_ALLOW},
			wantCalls: []models.TransferCheck{expectedTransfer(TEST_AMOUNT, "RUB")},
			want:      &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_ALLOW},
		},
		{
			name:      "allow with empty hits",
			req:       validCheckRequest(TEST_AMOUNT, "EUR"),
			check:     &models.Check{ID: checkID, Verdict: models.VERDICT_ALLOW, Hits: []models.Hit{}},
			wantCalls: []models.TransferCheck{expectedTransfer(TEST_AMOUNT, "EUR")},
			want:      &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_ALLOW},
		},
		{
			name:      "deny with single hit",
			req:       validCheckRequest(TEST_AMOUNT, "usd"),
			check:     &models.Check{ID: checkID, Verdict: models.VERDICT_DENY, Hits: []models.Hit{{RuleCode: "single_amount_limit", Reason: "over"}}},
			wantCalls: []models.TransferCheck{expectedTransfer(TEST_AMOUNT, "USD")},
			want: &antifraudv1.CheckTransferResponse{
				CheckId: TEST_CHECK_ID,
				Verdict: antifraudv1.Verdict_VERDICT_DENY,
				Reasons: []string{"single_amount_limit: over"},
			},
		},
		{
			name:      "currency with spaces is trimmed",
			req:       validCheckRequest(TEST_AMOUNT, " usd "),
			check:     &models.Check{ID: checkID, Verdict: models.VERDICT_ALLOW},
			wantCalls: []models.TransferCheck{expectedTransfer(TEST_AMOUNT, "USD")},
			want:      &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_ALLOW},
		},
		{
			name:      "zero amount passed as is",
			req:       validCheckRequest(0, "RUB"),
			check:     &models.Check{ID: checkID, Verdict: models.VERDICT_ALLOW},
			wantCalls: []models.TransferCheck{expectedTransfer(0, "RUB")},
			want:      &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_ALLOW},
		},
		{
			name:      "negative amount passed as is",
			req:       validCheckRequest(-5, "RUB"),
			check:     &models.Check{ID: checkID, Verdict: models.VERDICT_ALLOW},
			wantCalls: []models.TransferCheck{expectedTransfer(-5, "RUB")},
			want:      &antifraudv1.CheckTransferResponse{CheckId: TEST_CHECK_ID, Verdict: antifraudv1.Verdict_VERDICT_ALLOW},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checks := &fakeCheckService{check: tt.check}
			srv := newTestServer(checks, &fakeRuleService{})

			got, err := srv.CheckTransfer(t.Context(), tt.req)

			require.NoError(t, err)
			assert.Equal(t, tt.wantCalls, checks.calls)
			assert.True(t, proto.Equal(tt.want, got), "want %v, got %v", tt.want, got)
			assert.Len(t, got.GetReasons(), len(tt.want.GetReasons()))
		})
	}
}

func TestCheckTransfer_InvalidIDs(t *testing.T) {
	tests := []struct {
		name    string
		req     *antifraudv1.CheckTransferRequest
		wantErr error
	}{
		{
			name:    "empty initiator_id",
			req:     &antifraudv1.CheckTransferRequest{InitiatorId: "", FromAccountId: TEST_FROM_ACCOUNT_ID, ToAccountId: TEST_TO_ACCOUNT_ID, Amount: TEST_AMOUNT, Currency: "RUB"},
			wantErr: domain.ErrInitiatorIDMustBeUUID,
		},
		{
			name:    "bad initiator_id",
			req:     &antifraudv1.CheckTransferRequest{InitiatorId: "bad", FromAccountId: TEST_FROM_ACCOUNT_ID, ToAccountId: TEST_TO_ACCOUNT_ID, Amount: TEST_AMOUNT, Currency: "RUB"},
			wantErr: domain.ErrInitiatorIDMustBeUUID,
		},
		{
			name:    "bad initiator_id reported before bad from_account_id",
			req:     &antifraudv1.CheckTransferRequest{InitiatorId: "bad", FromAccountId: "bad", ToAccountId: TEST_TO_ACCOUNT_ID, Amount: TEST_AMOUNT, Currency: "RUB"},
			wantErr: domain.ErrInitiatorIDMustBeUUID,
		},
		{
			name:    "empty from_account_id",
			req:     &antifraudv1.CheckTransferRequest{InitiatorId: TEST_INITIATOR_ID, FromAccountId: "", ToAccountId: TEST_TO_ACCOUNT_ID, Amount: TEST_AMOUNT, Currency: "RUB"},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name:    "bad from_account_id reported before bad currency",
			req:     &antifraudv1.CheckTransferRequest{InitiatorId: TEST_INITIATOR_ID, FromAccountId: "bad", ToAccountId: TEST_TO_ACCOUNT_ID, Amount: TEST_AMOUNT, Currency: "xx"},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name:    "bad to_account_id",
			req:     &antifraudv1.CheckTransferRequest{InitiatorId: TEST_INITIATOR_ID, FromAccountId: TEST_FROM_ACCOUNT_ID, ToAccountId: "bad", Amount: TEST_AMOUNT, Currency: "RUB"},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
		{
			name:    "bad to_account_id reported before bad currency",
			req:     &antifraudv1.CheckTransferRequest{InitiatorId: TEST_INITIATOR_ID, FromAccountId: TEST_FROM_ACCOUNT_ID, ToAccountId: "bad", Amount: TEST_AMOUNT, Currency: "xx"},
			wantErr: domain.ErrAccountIDMustBeUUID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checks := &fakeCheckService{}
			srv := newTestServer(checks, &fakeRuleService{})

			got, err := srv.CheckTransfer(t.Context(), tt.req)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Nil(t, got)
			assert.Empty(t, checks.calls)
		})
	}
}

func TestCheckTransfer_InvalidCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
	}{
		{name: "empty", currency: ""},
		{name: "two letters", currency: "RU"},
		{name: "four letters", currency: "RUBL"},
		{name: "digit inside", currency: "R1B"},
		{name: "cyrillic", currency: "руб"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checks := &fakeCheckService{}
			srv := newTestServer(checks, &fakeRuleService{})

			got, err := srv.CheckTransfer(t.Context(), validCheckRequest(TEST_AMOUNT, tt.currency))

			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Nil(t, got)
			assert.Empty(t, checks.calls)
		})
	}
}

func TestCheckTransfer_ServiceError(t *testing.T) {
	checks := &fakeCheckService{err: fmt.Errorf("record check: %w", errDB)}
	srv := newTestServer(checks, &fakeRuleService{})

	got, err := srv.CheckTransfer(t.Context(), validCheckRequest(TEST_AMOUNT, "RUB"))

	require.ErrorIs(t, err, domain.ErrCheckFailed)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Equal(t, "failed to check transfer", status.Convert(err).Message())
	assert.NotContains(t, status.Convert(err).Message(), TEST_SECRET)
	assert.Nil(t, got)
	assert.Equal(t, []models.TransferCheck{expectedTransfer(TEST_AMOUNT, "RUB")}, checks.calls)
}

// sampleRules - три правила с разными параметрами для ListRules
func sampleRules() []models.RuleConfig {
	return []models.RuleConfig{
		{Code: "velocity", Description: "transfers per window", Enabled: true, Params: models.RuleParams{MaxCount: 3, WindowSeconds: 60}, UpdatedAt: testUpdatedAt()},
		{Code: "single_amount_limit", Description: "single limit", Enabled: false, Params: models.RuleParams{Limits: map[models.Currency]int64{"RUB": 1000, "USD": 50}}, UpdatedAt: testUpdatedAt()},
		{Code: "same_account", Description: "same account", Enabled: true, UpdatedBy: uuidPtr(TEST_OFFICER_ID), UpdatedAt: testUpdatedAt()},
	}
}

func TestListRules(t *testing.T) {
	tests := []struct {
		name       string
		listResult []models.RuleConfig
		want       []*antifraudv1.Rule
	}{
		{
			name:       "three rules in service order",
			listResult: sampleRules(),
			want: []*antifraudv1.Rule{
				{Code: "velocity", Description: "transfers per window", Enabled: true, Params: &antifraudv1.RuleParams{Limits: map[string]int64{}, MaxCount: 3, WindowSeconds: 60}, UpdatedBy: "", UpdatedAt: timestamppb.New(testUpdatedAt())},
				{Code: "single_amount_limit", Description: "single limit", Enabled: false, Params: &antifraudv1.RuleParams{Limits: map[string]int64{"RUB": 1000, "USD": 50}}, UpdatedBy: "", UpdatedAt: timestamppb.New(testUpdatedAt())},
				{Code: "same_account", Description: "same account", Enabled: true, Params: &antifraudv1.RuleParams{Limits: map[string]int64{}}, UpdatedBy: TEST_OFFICER_ID, UpdatedAt: timestamppb.New(testUpdatedAt())},
			},
		},
		{name: "nil list", listResult: nil, want: nil},
		{name: "empty list", listResult: []models.RuleConfig{}, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := &fakeRuleService{listResult: tt.listResult}
			srv := newTestServer(&fakeCheckService{}, rules)

			got, err := srv.ListRules(t.Context(), &antifraudv1.ListRulesRequest{})

			require.NoError(t, err)
			require.Len(t, got.GetRules(), len(tt.want))
			for i := range tt.want {
				assert.True(t, proto.Equal(tt.want[i], got.GetRules()[i]), "rule %d: want %v, got %v", i, tt.want[i], got.GetRules()[i])
				assert.True(t, proto.Equal(ruleToProto(&tt.listResult[i]), got.GetRules()[i]), "rule %d differs from ruleToProto", i)
			}
			assert.Equal(t, 1, rules.listCalls)
		})
	}
}

func TestListRules_ServiceError(t *testing.T) {
	rules := &fakeRuleService{listErr: fmt.Errorf("select rules: %w", errDB)}
	srv := newTestServer(&fakeCheckService{}, rules)

	got, err := srv.ListRules(t.Context(), &antifraudv1.ListRulesRequest{})

	require.ErrorIs(t, err, domain.ErrListRulesFailed)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Equal(t, "failed to list rules", status.Convert(err).Message())
	assert.NotContains(t, status.Convert(err).Message(), TEST_SECRET)
	assert.Nil(t, got)
}

func TestUpdateRule_Success(t *testing.T) {
	officerID := uuid.MustParse(TEST_OFFICER_ID)
	result := &models.RuleConfig{
		Code:        "velocity",
		Description: "transfers per window",
		Enabled:     true,
		Params:      models.RuleParams{Limits: map[models.Currency]int64{"RUB": 1000}, MaxCount: 5, WindowSeconds: 60},
		UpdatedBy:   &officerID,
		UpdatedAt:   testUpdatedAt(),
	}
	wantResponse := &antifraudv1.Rule{
		Code:        "velocity",
		Description: "transfers per window",
		Enabled:     true,
		Params:      &antifraudv1.RuleParams{Limits: map[string]int64{"RUB": 1000}, MaxCount: 5, WindowSeconds: 60},
		UpdatedBy:   TEST_OFFICER_ID,
		UpdatedAt:   timestamppb.New(testUpdatedAt()),
	}

	tests := []struct {
		name      string
		req       *antifraudv1.UpdateRuleRequest
		wantCalls []models.RuleUpdate
	}{
		{
			name: "params only, enabled not set",
			req:  &antifraudv1.UpdateRuleRequest{Code: "velocity", Params: &antifraudv1.RuleParams{MaxCount: 5, WindowSeconds: 60}, OfficerId: TEST_OFFICER_ID},
			wantCalls: []models.RuleUpdate{
				{Code: "velocity", Enabled: nil, Params: &models.RuleParams{MaxCount: 5, WindowSeconds: 60}, OfficerID: officerID},
			},
		},
		{
			name: "enabled false and limits normalized",
			req:  &antifraudv1.UpdateRuleRequest{Code: "single_amount_limit", Enabled: boolPtr(false), Params: &antifraudv1.RuleParams{Limits: map[string]int64{"rub": 1000}}, OfficerId: TEST_OFFICER_ID},
			wantCalls: []models.RuleUpdate{
				{Code: "single_amount_limit", Enabled: boolPtr(false), Params: &models.RuleParams{Limits: map[models.Currency]int64{"RUB": 1000}}, OfficerID: officerID},
			},
		},
		{
			name: "enabled true without params",
			req:  &antifraudv1.UpdateRuleRequest{Code: "velocity", Enabled: boolPtr(true), OfficerId: TEST_OFFICER_ID},
			wantCalls: []models.RuleUpdate{
				{Code: "velocity", Enabled: boolPtr(true), Params: nil, OfficerID: officerID},
			},
		},
		{
			name: "nothing set is passed to service",
			req:  &antifraudv1.UpdateRuleRequest{Code: "velocity", OfficerId: TEST_OFFICER_ID},
			wantCalls: []models.RuleUpdate{
				{Code: "velocity", Enabled: nil, Params: nil, OfficerID: officerID},
			},
		},
		{
			name: "empty params are not nil",
			req:  &antifraudv1.UpdateRuleRequest{Code: "velocity", Params: &antifraudv1.RuleParams{}, OfficerId: TEST_OFFICER_ID},
			wantCalls: []models.RuleUpdate{
				{Code: "velocity", Enabled: nil, Params: &models.RuleParams{}, OfficerID: officerID},
			},
		},
		{
			name: "zero and negative limits passed as is",
			req:  &antifraudv1.UpdateRuleRequest{Code: "daily_amount_limit", Params: &antifraudv1.RuleParams{Limits: map[string]int64{" usd ": 0, "eur": -1}}, OfficerId: TEST_OFFICER_ID},
			wantCalls: []models.RuleUpdate{
				{Code: "daily_amount_limit", Enabled: nil, Params: &models.RuleParams{Limits: map[models.Currency]int64{"USD": 0, "EUR": -1}}, OfficerID: officerID},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := &fakeRuleService{updateResult: result}
			srv := newTestServer(&fakeCheckService{}, rules)

			got, err := srv.UpdateRule(t.Context(), tt.req)

			require.NoError(t, err)
			assert.Equal(t, normalizeUpdates(tt.wantCalls), normalizeUpdates(rules.updateCalls))
			assert.True(t, proto.Equal(wantResponse, got), "want %v, got %v", wantResponse, got)
		})
	}
}

func TestUpdateRule_InvalidOfficerID(t *testing.T) {
	tests := []struct {
		name      string
		officerID string
	}{
		{name: "empty officer_id", officerID: ""},
		{name: "bad officer_id", officerID: "bad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := &fakeRuleService{}
			srv := newTestServer(&fakeCheckService{}, rules)

			got, err := srv.UpdateRule(t.Context(), &antifraudv1.UpdateRuleRequest{Code: "velocity", Enabled: boolPtr(true), OfficerId: tt.officerID})

			require.ErrorIs(t, err, domain.ErrOfficerIDMustBeUUID)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Nil(t, got)
			assert.Empty(t, rules.updateCalls)
		})
	}
}

func TestUpdateRule_InvalidCurrency(t *testing.T) {
	tests := []struct {
		name   string
		limits map[string]int64
	}{
		{name: "digit in code", limits: map[string]int64{"xq1": 1}},
		{name: "empty code", limits: map[string]int64{"": 1}},
		{name: "valid and invalid code", limits: map[string]int64{"RUB": 1, "RUBL": 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := &fakeRuleService{}
			srv := newTestServer(&fakeCheckService{}, rules)

			got, err := srv.UpdateRule(t.Context(), &antifraudv1.UpdateRuleRequest{Code: "single_amount_limit", Params: &antifraudv1.RuleParams{Limits: tt.limits}, OfficerId: TEST_OFFICER_ID})

			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Nil(t, got)
			assert.Empty(t, rules.updateCalls)
		})
	}
}

func TestUpdateRule_ServiceErrors(t *testing.T) {
	tests := []struct {
		name     string
		svcErr   error
		wantCode codes.Code
		wantMsg  string
	}{
		{
			name:     "wrapped rule not found",
			svcErr:   fmt.Errorf("update velocity: %w", domain.ErrRuleNotFound),
			wantCode: codes.NotFound,
			wantMsg:  "rule not found",
		},
		{
			name:     "bare rule not found",
			svcErr:   domain.ErrRuleNotFound,
			wantCode: codes.NotFound,
			wantMsg:  "rule not found",
		},
		{
			name:     "invalid params message goes to officer",
			svcErr:   fmt.Errorf("%w: velocity: max_count must be positive", domain.ErrInvalidRuleParams),
			wantCode: codes.InvalidArgument,
			wantMsg:  "invalid rule params: velocity: max_count must be positive",
		},
		{
			name:     "db error hidden",
			svcErr:   fmt.Errorf("update rule: %w", errDB),
			wantCode: codes.Internal,
			wantMsg:  "failed to update rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := &fakeRuleService{updateErr: tt.svcErr}
			srv := newTestServer(&fakeCheckService{}, rules)

			got, err := srv.UpdateRule(t.Context(), &antifraudv1.UpdateRuleRequest{Code: "velocity", Enabled: boolPtr(true), OfficerId: TEST_OFFICER_ID})

			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.Equal(t, tt.wantMsg, status.Convert(err).Message())
			assert.Nil(t, got)
			assert.Len(t, rules.updateCalls, 1)
		})
	}
}

func TestUpdateRule_ServiceErrorSentinels(t *testing.T) {
	tests := []struct {
		name    string
		svcErr  error
		wantErr error
	}{
		{name: "rule not found maps to ErrRuleMissing", svcErr: fmt.Errorf("update velocity: %w", domain.ErrRuleNotFound), wantErr: domain.ErrRuleMissing},
		{name: "db error maps to ErrUpdateRuleFailed", svcErr: fmt.Errorf("update rule: %w", errDB), wantErr: domain.ErrUpdateRuleFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := &fakeRuleService{updateErr: tt.svcErr}
			srv := newTestServer(&fakeCheckService{}, rules)

			got, err := srv.UpdateRule(t.Context(), &antifraudv1.UpdateRuleRequest{Code: "velocity", Enabled: boolPtr(true), OfficerId: TEST_OFFICER_ID})

			require.ErrorIs(t, err, tt.wantErr)
			assert.NotContains(t, status.Convert(err).Message(), TEST_SECRET)
			assert.Nil(t, got)
		})
	}
}

// invalidCurrencyInputs - неверные коды валют от клиента для R11
func invalidCurrencyInputs() []string {
	return []string{"xq1", "Zz", "ABCD"}
}

// R11: сообщение об ошибке валюты в CheckTransfer не содержит ввода клиента
func TestCheckTransfer_InvalidCurrencyNotEchoed(t *testing.T) {
	for _, currency := range invalidCurrencyInputs() {
		t.Run(currency, func(t *testing.T) {
			checks := &fakeCheckService{}
			srv := newTestServer(checks, &fakeRuleService{})

			got, err := srv.CheckTransfer(t.Context(), validCheckRequest(TEST_AMOUNT, currency))

			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.NotContains(t, status.Convert(err).Message(), currency)
			assert.NotContains(t, status.Convert(err).Message(), strings.ToUpper(currency))
			assert.Nil(t, got)
		})
	}
}

// R11: сообщение об ошибке ключа limits в UpdateRule не содержит ввода клиента
func TestUpdateRule_InvalidCurrencyNotEchoed(t *testing.T) {
	for _, currency := range invalidCurrencyInputs() {
		t.Run(currency, func(t *testing.T) {
			rules := &fakeRuleService{}
			srv := newTestServer(&fakeCheckService{}, rules)

			got, err := srv.UpdateRule(t.Context(), &antifraudv1.UpdateRuleRequest{Code: "single_amount_limit", Params: &antifraudv1.RuleParams{Limits: map[string]int64{currency: 1}}, OfficerId: TEST_OFFICER_ID})

			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.NotContains(t, status.Convert(err).Message(), currency)
			assert.NotContains(t, status.Convert(err).Message(), strings.ToUpper(currency))
			assert.Nil(t, got)
		})
	}
}

// R12: дубль валюты после нормализации отвергается до вызова сервиса
func TestUpdateRule_DuplicateCurrency(t *testing.T) {
	// результат задан, чтобы на текущем коде тест падал проверкой, а не паникой
	rules := &fakeRuleService{updateResult: &models.RuleConfig{Code: "single_amount_limit", UpdatedAt: testUpdatedAt()}}
	srv := newTestServer(&fakeCheckService{}, rules)

	got, err := srv.UpdateRule(t.Context(), &antifraudv1.UpdateRuleRequest{Code: "single_amount_limit", Params: &antifraudv1.RuleParams{Limits: map[string]int64{"rub": 1, "RUB": 2}}, OfficerId: TEST_OFFICER_ID})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Nil(t, got)
	assert.Empty(t, rules.updateCalls)
}
