package rules

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

const TEST_UNKNOWN_RULE = "unknown"

// ruleCodes - коды правил в порядке среза.
func ruleCodes(rules []Rule) []string {
	codes := make([]string, 0, len(rules))
	for _, rule := range rules {
		codes = append(codes, rule.Code())
	}

	return codes
}

func TestBuildRules(t *testing.T) {
	rubLimits := models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: TEST_LIMIT}}
	velocityParams := models.RuleParams{MaxCount: 2, WindowSeconds: 120}

	tests := []struct {
		name      string
		configs   []models.RuleConfig
		wantCodes []string
		wantErr   bool
	}{
		{name: "nil configs", configs: nil, wantCodes: []string{}},
		{name: "empty configs", configs: []models.RuleConfig{}, wantCodes: []string{}},
		{
			name:      "disabled unknown rule skipped",
			configs:   []models.RuleConfig{{Code: TEST_UNKNOWN_RULE, Enabled: false}},
			wantCodes: []string{},
		},
		{
			name:    "enabled unknown rule",
			configs: []models.RuleConfig{{Code: TEST_UNKNOWN_RULE, Enabled: true}},
			wantErr: true,
		},
		{
			name: "valid rule followed by enabled unknown rule",
			configs: []models.RuleConfig{
				{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Enabled: true, Params: rubLimits},
				{Code: TEST_UNKNOWN_RULE, Enabled: true},
			},
			wantErr: true,
		},
		{
			name:    "enabled empty code",
			configs: []models.RuleConfig{{Code: "", Enabled: true}},
			wantErr: true,
		},
		{
			name: "all enabled keep config order",
			configs: []models.RuleConfig{
				{Code: domain.RULE_SAME_ACCOUNT, Enabled: true},
				{Code: domain.RULE_VELOCITY, Enabled: true, Params: velocityParams},
				{Code: domain.RULE_DAILY_AMOUNT_LIMIT, Enabled: true, Params: rubLimits},
				{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Enabled: true, Params: rubLimits},
			},
			wantCodes: []string{domain.RULE_SAME_ACCOUNT, domain.RULE_VELOCITY, domain.RULE_DAILY_AMOUNT_LIMIT, domain.RULE_SINGLE_AMOUNT_LIMIT},
		},
		{
			name: "disabled rule in the middle skipped",
			configs: []models.RuleConfig{
				{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Enabled: true, Params: rubLimits},
				{Code: domain.RULE_VELOCITY, Enabled: false, Params: velocityParams},
				{Code: domain.RULE_DAILY_AMOUNT_LIMIT, Enabled: true, Params: rubLimits},
			},
			wantCodes: []string{domain.RULE_SINGLE_AMOUNT_LIMIT, domain.RULE_DAILY_AMOUNT_LIMIT},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, err := Build(tt.configs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, rules)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantCodes, ruleCodes(rules))
		})
	}
}

func TestBuildRulesUseParams(t *testing.T) {
	rubLimits := models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: TEST_LIMIT}}
	velocityParams := models.RuleParams{MaxCount: 2, WindowSeconds: 120}
	testDay := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	// 15:30Z - 120 с
	velocitySince := time.Date(2026, 9, 28, 15, 28, 0, 0, time.UTC)

	tests := []struct {
		name           string
		config         models.RuleConfig
		amount         int64
		allowed        int
		dailyTotal     int64
		wantHit        bool
		wantCountCalls []countAllowedCall
		wantDailyCalls []dailyTotalCall
	}{
		{
			name:   "single limit not exceeded",
			config: models.RuleConfig{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Enabled: true, Params: rubLimits},
			amount: 1000, wantHit: false,
		},
		{
			name:   "single limit exceeded",
			config: models.RuleConfig{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Enabled: true, Params: rubLimits},
			amount: 1001, wantHit: true,
		},
		{
			name:   "daily limit reached exactly",
			config: models.RuleConfig{Code: domain.RULE_DAILY_AMOUNT_LIMIT, Enabled: true, Params: rubLimits},
			amount: 500, dailyTotal: 500, wantHit: false,
			wantDailyCalls: []dailyTotalCall{{AccountID: TEST_FROM_ACCOUNT_ID, Currency: TEST_RUB, Day: testDay}},
		},
		{
			name:   "daily limit exceeded",
			config: models.RuleConfig{Code: domain.RULE_DAILY_AMOUNT_LIMIT, Enabled: true, Params: rubLimits},
			amount: 501, dailyTotal: 500, wantHit: true,
			wantDailyCalls: []dailyTotalCall{{AccountID: TEST_FROM_ACCOUNT_ID, Currency: TEST_RUB, Day: testDay}},
		},
		{
			name:   "velocity below max",
			config: models.RuleConfig{Code: domain.RULE_VELOCITY, Enabled: true, Params: velocityParams},
			amount: 100, allowed: 1, wantHit: false,
			wantCountCalls: []countAllowedCall{{AccountID: TEST_FROM_ACCOUNT_ID, Since: velocitySince}},
		},
		{
			name:   "velocity at max",
			config: models.RuleConfig{Code: domain.RULE_VELOCITY, Enabled: true, Params: velocityParams},
			amount: 100, allowed: 2, wantHit: true,
			wantCountCalls: []countAllowedCall{{AccountID: TEST_FROM_ACCOUNT_ID, Since: velocitySince}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &recordingStats{allowed: tt.allowed, dailyTotal: tt.dailyTotal}

			rules, err := Build([]models.RuleConfig{tt.config})
			require.NoError(t, err)
			require.Len(t, rules, 1)

			reason, err := rules[0].Check(t.Context(), fixedTransfer(tt.amount, TEST_RUB), stats)

			require.NoError(t, err)
			assert.Equal(t, tt.wantHit, reason != "")
			assert.Equal(t, tt.wantCountCalls, stats.countCalls)
			assert.Equal(t, tt.wantDailyCalls, stats.dailyCalls)
		})
	}
}

type validateCase struct {
	name    string
	code    string
	params  models.RuleParams
	wantErr error
}

// limitRuleCases - одинаковые сценарии для правил с порогами сумм.
func limitRuleCases(code string) []validateCase {
	return []validateCase{
		{code + " single valid limit", code, models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 1}}, nil},
		{code + " two valid limits", code, models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 1, TEST_USD: 5}}, nil},
		{code + " zero params", code, models.RuleParams{}, nil},
		{code + " empty non nil limits", code, models.RuleParams{Limits: map[models.Currency]int64{}}, nil},
		{code + " zero limit", code, models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 0}}, domain.ErrInvalidRuleParams},
		{code + " negative limit", code, models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: -5}}, domain.ErrInvalidRuleParams},
		{code + " currency not in set", code, models.RuleParams{Limits: map[models.Currency]int64{"GEL": 1}}, domain.ErrInvalidRuleParams},
		{code + " lowercase currency", code, models.RuleParams{Limits: map[models.Currency]int64{"rub": 1}}, domain.ErrInvalidRuleParams},
		{code + " one bad currency of two", code, models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 1, "GEL": 1}}, domain.ErrInvalidRuleParams},
		{code + " max count set", code, models.RuleParams{MaxCount: 1}, domain.ErrInvalidRuleParams},
		{code + " negative max count", code, models.RuleParams{MaxCount: -1}, domain.ErrInvalidRuleParams},
		{code + " negative window", code, models.RuleParams{WindowSeconds: -1}, domain.ErrInvalidRuleParams},
		{code + " window with valid limits", code, models.RuleParams{WindowSeconds: 60, Limits: map[models.Currency]int64{TEST_RUB: 1}}, domain.ErrInvalidRuleParams},
	}
}

func TestValidateParams(t *testing.T) {
	currencies, err := models.NewCurrencySet([]string{"RUB", "USD"})
	require.NoError(t, err)

	tests := append(limitRuleCases(domain.RULE_SINGLE_AMOUNT_LIMIT), limitRuleCases(domain.RULE_DAILY_AMOUNT_LIMIT)...)
	tests = append(tests, []validateCase{
		{"velocity valid", domain.RULE_VELOCITY, models.RuleParams{MaxCount: 5, WindowSeconds: 60}, nil},
		{"velocity minimal", domain.RULE_VELOCITY, models.RuleParams{MaxCount: 1, WindowSeconds: 1}, nil},
		{"velocity empty non nil limits", domain.RULE_VELOCITY, models.RuleParams{MaxCount: 5, WindowSeconds: 60, Limits: map[models.Currency]int64{}}, nil},
		{"velocity zero max count", domain.RULE_VELOCITY, models.RuleParams{MaxCount: 0, WindowSeconds: 60}, domain.ErrInvalidRuleParams},
		{"velocity zero window", domain.RULE_VELOCITY, models.RuleParams{MaxCount: 5, WindowSeconds: 0}, domain.ErrInvalidRuleParams},
		{"velocity negative max count", domain.RULE_VELOCITY, models.RuleParams{MaxCount: -1, WindowSeconds: 60}, domain.ErrInvalidRuleParams},
		{"velocity negative window", domain.RULE_VELOCITY, models.RuleParams{MaxCount: 5, WindowSeconds: -1}, domain.ErrInvalidRuleParams},
		{"velocity zero params", domain.RULE_VELOCITY, models.RuleParams{}, domain.ErrInvalidRuleParams},
		{"velocity with limits", domain.RULE_VELOCITY, models.RuleParams{MaxCount: 5, WindowSeconds: 60, Limits: map[models.Currency]int64{TEST_RUB: 1}}, domain.ErrInvalidRuleParams},
		{"same account zero params", domain.RULE_SAME_ACCOUNT, models.RuleParams{}, nil},
		{"same account empty non nil limits", domain.RULE_SAME_ACCOUNT, models.RuleParams{Limits: map[models.Currency]int64{}}, nil},
		{"same account max count", domain.RULE_SAME_ACCOUNT, models.RuleParams{MaxCount: 1}, domain.ErrInvalidRuleParams},
		{"same account window", domain.RULE_SAME_ACCOUNT, models.RuleParams{WindowSeconds: 1}, domain.ErrInvalidRuleParams},
		{"same account negative max count", domain.RULE_SAME_ACCOUNT, models.RuleParams{MaxCount: -1}, domain.ErrInvalidRuleParams},
		{"same account limits", domain.RULE_SAME_ACCOUNT, models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 1}}, domain.ErrInvalidRuleParams},
		{"unknown code", TEST_UNKNOWN_RULE, models.RuleParams{}, domain.ErrRuleNotFound},
		{"empty code", "", models.RuleParams{}, domain.ErrRuleNotFound},
		{"uppercase code", "VELOCITY", models.RuleParams{MaxCount: 5, WindowSeconds: 60}, domain.ErrRuleNotFound},
	}...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.code, tt.params, currencies)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

// newStubRules - свежие заглушки с кодами rule_0.. и заданными причинами.
func newStubRules(reasons []string) ([]*stubRule, []Rule) {
	stubs := make([]*stubRule, 0, len(reasons))
	rules := make([]Rule, 0, len(reasons))
	for i, reason := range reasons {
		stub := &stubRule{code: "rule_" + string(rune('0'+i)), reason: reason}
		stubs = append(stubs, stub)
		rules = append(rules, stub)
	}

	return stubs, rules
}

func TestEvaluateHits(t *testing.T) {
	tests := []struct {
		name    string
		reasons []string
		want    []models.Hit
	}{
		{name: "no rules", reasons: nil, want: []models.Hit{}},
		{name: "no hits", reasons: []string{"", "", ""}, want: []models.Hit{}},
		{name: "middle hit", reasons: []string{"", "r-b", ""}, want: []models.Hit{{RuleCode: "rule_1", Reason: "r-b"}}},
		{
			name:    "all hit",
			reasons: []string{"r-a", "r-b", "r-c"},
			want:    []models.Hit{{RuleCode: "rule_0", Reason: "r-a"}, {RuleCode: "rule_1", Reason: "r-b"}, {RuleCode: "rule_2", Reason: "r-c"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubs, rules := newStubRules(tt.reasons)
			stats := &recordingStats{}
			transfer := fixedTransfer(100, TEST_RUB)

			hits, err := Evaluate(t.Context(), rules, transfer, stats)

			require.NoError(t, err)
			// nil и пустой срез спецификация не различает
			assert.Equal(t, tt.want, append([]models.Hit{}, hits...))
			for _, stub := range stubs {
				assert.Equal(t, 1, stub.calls)
				assert.Equal(t, []models.TransferCheck{transfer}, stub.transfers)
				require.Len(t, stub.stats, 1)
				assert.Same(t, stats, stub.stats[0])
			}
		})
	}
}

func TestEvaluateRealRules(t *testing.T) {
	rules, err := Build([]models.RuleConfig{
		{Code: domain.RULE_SINGLE_AMOUNT_LIMIT, Enabled: true, Params: models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 10}}},
		{Code: domain.RULE_SAME_ACCOUNT, Enabled: true},
		{Code: domain.RULE_VELOCITY, Enabled: true, Params: models.RuleParams{MaxCount: 1, WindowSeconds: 60}},
	})
	require.NoError(t, err)
	stats := &recordingStats{allowed: 5}

	hits, err := Evaluate(t.Context(), rules, fixedTransfer(100, TEST_RUB), stats)

	require.NoError(t, err)
	require.Len(t, hits, 2)
	assert.Equal(t, domain.RULE_SINGLE_AMOUNT_LIMIT, hits[0].RuleCode)
	assert.NotEqual(t, "", hits[0].Reason)
	assert.Equal(t, domain.RULE_VELOCITY, hits[1].RuleCode)
	assert.NotEqual(t, "", hits[1].Reason)
}

func TestEvaluateRuleError(t *testing.T) {
	errRule := errors.New("rule failure")
	errDB := errors.New("db failure")

	tests := []struct {
		name     string
		newRules func() []Rule
		stats    func() interfaces.CheckStats
		wantErr  error
	}{
		{
			name: "middle rule fails",
			newRules: func() []Rule {
				return []Rule{&stubRule{code: "a"}, &stubRule{code: "b", err: errRule}, &stubRule{code: "c"}}
			},
			stats:   func() interfaces.CheckStats { return &recordingStats{} },
			wantErr: errRule,
		},
		{
			name: "first rule fails",
			newRules: func() []Rule {
				return []Rule{&stubRule{code: "a", err: errRule}, &stubRule{code: "b", reason: "r-b"}}
			},
			stats:   func() interfaces.CheckStats { return &recordingStats{} },
			wantErr: errRule,
		},
		{
			name: "hit then last rule fails",
			newRules: func() []Rule {
				return []Rule{&stubRule{code: "a", reason: "r-a"}, &stubRule{code: "b", err: errRule}}
			},
			stats:   func() interfaces.CheckStats { return &recordingStats{} },
			wantErr: errRule,
		},
		{
			name:     "real velocity stats error",
			newRules: func() []Rule { return []Rule{newVelocity(models.RuleParams{MaxCount: 3, WindowSeconds: 60})} },
			stats:    func() interfaces.CheckStats { return &recordingStats{countErr: errDB} },
			wantErr:  errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits, err := Evaluate(t.Context(), tt.newRules(), fixedTransfer(100, TEST_RUB), tt.stats())

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, hits)
		})
	}
}

func TestEvaluateHitsInvariant(t *testing.T) {
	tests := []struct {
		name    string
		reasons []string
		want    []models.Hit
	}{
		{name: "none of five", reasons: []string{"", "", "", "", ""}, want: []models.Hit{}},
		{name: "one of five", reasons: []string{"", "", "r-2", "", ""}, want: []models.Hit{{RuleCode: "rule_2", Reason: "r-2"}}},
		{
			name:    "all of four",
			reasons: []string{"r-0", "r-1", "r-2", "r-3"},
			want: []models.Hit{
				{RuleCode: "rule_0", Reason: "r-0"}, {RuleCode: "rule_1", Reason: "r-1"},
				{RuleCode: "rule_2", Reason: "r-2"}, {RuleCode: "rule_3", Reason: "r-3"},
			},
		},
		{
			name:    "first and last",
			reasons: []string{"r-0", "", "", "r-3"},
			want:    []models.Hit{{RuleCode: "rule_0", Reason: "r-0"}, {RuleCode: "rule_3", Reason: "r-3"}},
		},
		{name: "single rule", reasons: []string{"r-0"}, want: []models.Hit{{RuleCode: "rule_0", Reason: "r-0"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, rules := newStubRules(tt.reasons)

			hits, err := Evaluate(t.Context(), rules, fixedTransfer(100, TEST_RUB), &recordingStats{})

			require.NoError(t, err)
			assert.LessOrEqual(t, len(hits), len(rules))
			// nil и пустой срез спецификация не различает
			assert.Equal(t, tt.want, append([]models.Hit{}, hits...))
		})
	}
}

func TestValidateImpliesBuildInvariant(t *testing.T) {
	currencies, err := models.NewCurrencySet([]string{"RUB", "USD"})
	require.NoError(t, err)

	tests := []struct {
		name   string
		code   string
		params models.RuleParams
	}{
		{"single with limit", domain.RULE_SINGLE_AMOUNT_LIMIT, models.RuleParams{Limits: map[models.Currency]int64{TEST_RUB: 1}}},
		{"single without limits", domain.RULE_SINGLE_AMOUNT_LIMIT, models.RuleParams{}},
		{"daily with limit", domain.RULE_DAILY_AMOUNT_LIMIT, models.RuleParams{Limits: map[models.Currency]int64{TEST_USD: 5}}},
		{"velocity", domain.RULE_VELOCITY, models.RuleParams{MaxCount: 5, WindowSeconds: 60}},
		{"same account", domain.RULE_SAME_ACCOUNT, models.RuleParams{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, Validate(tt.code, tt.params, currencies))

			rules, err := Build([]models.RuleConfig{{Code: tt.code, Enabled: true, Params: tt.params}})

			require.NoError(t, err)
			assert.Equal(t, []string{tt.code}, ruleCodes(rules))
		})
	}
}
