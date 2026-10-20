package server

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/pb/gen/antifraud/v1"
)

func TestRuleToProto(t *testing.T) {
	// тот же момент, что testUpdatedAt, но в зоне UTC+3
	moscowUpdatedAt := testUpdatedAt().In(time.FixedZone("MSK", 3*60*60))

	tests := []struct {
		name string
		rule models.RuleConfig
		want *antifraudv1.Rule
	}{
		{
			name: "velocity without updater",
			rule: models.RuleConfig{Code: "velocity", Description: "d", Enabled: true, Params: models.RuleParams{MaxCount: 3, WindowSeconds: 60}, UpdatedAt: testUpdatedAt()},
			want: &antifraudv1.Rule{Code: "velocity", Description: "d", Enabled: true, Params: &antifraudv1.RuleParams{Limits: map[string]int64{}, MaxCount: 3, WindowSeconds: 60}, UpdatedBy: "", UpdatedAt: timestamppb.New(testUpdatedAt())},
		},
		{
			name: "limits and updater",
			rule: models.RuleConfig{Code: "single_amount_limit", Description: "limit", Enabled: false, Params: models.RuleParams{Limits: map[models.Currency]int64{"RUB": 1000, "USD": 50}}, UpdatedBy: uuidPtr(TEST_OFFICER_ID), UpdatedAt: testUpdatedAt()},
			want: &antifraudv1.Rule{Code: "single_amount_limit", Description: "limit", Enabled: false, Params: &antifraudv1.RuleParams{Limits: map[string]int64{"RUB": 1000, "USD": 50}}, UpdatedBy: TEST_OFFICER_ID, UpdatedAt: timestamppb.New(testUpdatedAt())},
		},
		{
			name: "empty non-nil limits",
			rule: models.RuleConfig{Code: "daily_amount_limit", Params: models.RuleParams{Limits: map[models.Currency]int64{}}, UpdatedAt: testUpdatedAt()},
			want: &antifraudv1.Rule{Code: "daily_amount_limit", Params: &antifraudv1.RuleParams{Limits: map[string]int64{}}, UpdatedAt: timestamppb.New(testUpdatedAt())},
		},
		{
			name: "zero and negative limits kept",
			rule: models.RuleConfig{Code: "daily_amount_limit", Params: models.RuleParams{Limits: map[models.Currency]int64{"EUR": 0, "GBP": -7}}, UpdatedAt: testUpdatedAt()},
			want: &antifraudv1.Rule{Code: "daily_amount_limit", Params: &antifraudv1.RuleParams{Limits: map[string]int64{"EUR": 0, "GBP": -7}}, UpdatedAt: timestamppb.New(testUpdatedAt())},
		},
		{
			name: "updated_at in other zone is the same instant",
			rule: models.RuleConfig{Code: "same_account", Enabled: true, UpdatedAt: moscowUpdatedAt},
			want: &antifraudv1.Rule{Code: "same_account", Enabled: true, Params: &antifraudv1.RuleParams{Limits: map[string]int64{}}, UpdatedAt: timestamppb.New(testUpdatedAt())},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ruleToProto(&tt.rule)

			require.NotNil(t, got)
			require.NotNil(t, got.GetParams())
			assert.True(t, proto.Equal(tt.want, got), "want %v, got %v", tt.want, got)
			// proto.Equal не различает nil и пустую map, а R9 требует не-nil
			assert.NotNil(t, got.GetParams().GetLimits())
			assert.Equal(t, tt.want.GetParams().GetLimits(), got.GetParams().GetLimits())
			assert.True(t, tt.rule.UpdatedAt.Equal(got.GetUpdatedAt().AsTime()), "updated_at: want %v, got %v", tt.rule.UpdatedAt, got.GetUpdatedAt().AsTime())
		})
	}
}

func TestParamsFromProto(t *testing.T) {
	tests := []struct {
		name string
		in   *antifraudv1.RuleParams
		want *models.RuleParams
	}{
		{name: "nil params", in: nil, want: nil},
		{
			name: "limits normalized",
			in:   &antifraudv1.RuleParams{Limits: map[string]int64{"rub": 1000, " usd ": 200}, MaxCount: 5, WindowSeconds: 60},
			want: &models.RuleParams{Limits: map[models.Currency]int64{"RUB": 1000, "USD": 200}, MaxCount: 5, WindowSeconds: 60},
		},
		{
			name: "empty params",
			in:   &antifraudv1.RuleParams{},
			want: &models.RuleParams{},
		},
		{
			name: "zero and negative limits kept",
			in:   &antifraudv1.RuleParams{Limits: map[string]int64{"EUR": 0, "gbp": -7}},
			want: &models.RuleParams{Limits: map[models.Currency]int64{"EUR": 0, "GBP": -7}},
		},
		{
			name: "negative counters kept",
			in:   &antifraudv1.RuleParams{MaxCount: -1, WindowSeconds: -1},
			want: &models.RuleParams{MaxCount: -1, WindowSeconds: -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := paramsFromProto(tt.in)

			require.NoError(t, err)
			assert.Equal(t, normalizeEmptyLimits(tt.want), normalizeEmptyLimits(got))
		})
	}
}

func TestParamsFromProto_EmptyLimitsLength(t *testing.T) {
	got, err := paramsFromProto(&antifraudv1.RuleParams{MaxCount: 2})

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Len(t, got.Limits, 0)
	assert.Equal(t, 2, got.MaxCount)
}

func TestParamsFromProto_InvalidCurrency(t *testing.T) {
	tests := []struct {
		name   string
		limits map[string]int64
	}{
		{name: "digit in code", limits: map[string]int64{"xq1": 1}},
		{name: "empty code", limits: map[string]int64{"": 1}},
		{name: "four letters", limits: map[string]int64{"RUBL": 1}},
		{name: "space inside", limits: map[string]int64{"r b": 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := paramsFromProto(&antifraudv1.RuleParams{Limits: tt.limits, MaxCount: 1})

			require.Error(t, err)
			assert.Nil(t, got)
		})
	}
}

func TestRuleToProto_PreservesLimits(t *testing.T) {
	tests := []struct {
		name   string
		limits map[models.Currency]int64
	}{
		{name: "nil limits", limits: nil},
		{name: "one limit", limits: map[models.Currency]int64{"RUB": 1}},
		{name: "three limits", limits: map[models.Currency]int64{"RUB": 1000, "USD": 50, "EUR": 0}},
		{name: "extreme values", limits: map[models.Currency]int64{"GBP": -3, "JPY": math.MaxInt64}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ruleToProto(&models.RuleConfig{Code: "single_amount_limit", Params: models.RuleParams{Limits: tt.limits}, UpdatedAt: testUpdatedAt()})

			require.NotNil(t, got.GetParams())
			assert.Len(t, got.GetParams().GetLimits(), len(tt.limits))
			for currency, limit := range tt.limits {
				assert.Equal(t, limit, got.GetParams().GetLimits()[string(currency)], "currency %s", currency)
			}
		})
	}
}

func TestParamsFromProto_PreservesCounters(t *testing.T) {
	tests := []struct {
		name string
		in   *antifraudv1.RuleParams
	}{
		{name: "empty", in: &antifraudv1.RuleParams{}},
		{name: "one limit", in: &antifraudv1.RuleParams{Limits: map[string]int64{"rub": 1}, MaxCount: 1, WindowSeconds: 1}},
		{name: "mixed case limits", in: &antifraudv1.RuleParams{Limits: map[string]int64{" usd ": 5, "EUR": 7}, MaxCount: 10, WindowSeconds: 3600}},
		{name: "three limits zero counters", in: &antifraudv1.RuleParams{Limits: map[string]int64{"gbp": 0, "jpy": -1, "cny": 2}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := paramsFromProto(tt.in)

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, int(tt.in.GetMaxCount()), got.MaxCount)
			assert.Equal(t, int(tt.in.GetWindowSeconds()), got.WindowSeconds)
			assert.Len(t, got.Limits, len(tt.in.GetLimits()))
		})
	}
}

// R12: дубль валюты после нормализации - ошибка, а не случайный выбор порога
func TestParamsFromProto_DuplicateCurrency(t *testing.T) {
	tests := []struct {
		name   string
		limits map[string]int64
	}{
		{name: "lower and upper", limits: map[string]int64{"rub": 1, "RUB": 2}},
		{name: "spaces and upper", limits: map[string]int64{" usd ": 1, "USD": 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := paramsFromProto(&antifraudv1.RuleParams{Limits: tt.limits})

			require.Error(t, err)
			assert.Nil(t, got)
		})
	}
}
